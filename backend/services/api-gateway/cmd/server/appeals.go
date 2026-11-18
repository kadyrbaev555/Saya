package main

import (
	"github.com/gin-gonic/gin"
	pb "saya/contracts/auth/v1"
	wp "saya/contracts/workflow/v1"
)

func registerAppeals(r *gin.Engine, service wp.AssessmentClient, auth pb.AuthClient, authenticate gin.HandlerFunc, fail func(*gin.Context, error)) {
	base := "/api/v1/appeals"
	moderator := func(c *gin.Context) {
		u, err := auth.Me(c.Request.Context(), &pb.Identity{UserId: c.GetString("user_id")})
		if err != nil {
			c.Abort()
			fail(c, err)
			return
		}
		if u.Role != "ADMIN" {
			c.AbortWithStatus(403)
			return
		}
		c.Set("moderator", true)
	}
	list := func(scope string) gin.HandlerFunc {
		return func(c *gin.Context) {
			v, err := service.Appeals(c.Request.Context(), &wp.AppealQuery{Scope: scope, UserId: c.GetString("user_id"), Moderator: c.GetBool("moderator")})
			if err != nil {
				fail(c, err)
				return
			}
			items := make([]gin.H, 0, len(v.Items))
			for _, a := range v.Items {
				row := gin.H{"id": a.Id, "description": a.Description, "card_ready": a.CardReady}
				if scope != "public" {
					row["state"] = a.State
					row["reported"] = a.Reported
				}
				items = append(items, row)
			}
			c.Header("Cache-Control", "no-store")
			c.JSON(200, gin.H{"items": items, "checkout_available": v.CheckoutAvailable})
		}
	}
	change := func(action string) gin.HandlerFunc {
		return func(c *gin.Context) {
			req := &wp.AppealChange{Action: action, Id: c.Param("id"), UserId: c.GetString("user_id"), Moderator: c.GetBool("moderator")}
			if action == "submit" {
				var body struct {
					Description string
					Consent     bool
				}
				if c.ShouldBindJSON(&body) != nil {
					c.Status(400)
					return
				}
				req.Description = body.Description
				req.Consent = body.Consent
			}
			_, err := service.MutateAppeal(c.Request.Context(), req)
			if err != nil {
				fail(c, err)
				return
			}
			c.Status(204)
		}
	}
	r.GET(base, list("public"))
	r.GET(base+"/mine", authenticate, list("mine"))
	r.GET(base+"/review", authenticate, moderator, list("review"))
	r.POST(base+"/:id/submit", authenticate, change("submit"))
	r.POST(base+"/:id/withdraw", authenticate, change("withdraw"))
	r.POST(base+"/:id/report", change("report"))
	for _, a := range []string{"approve", "decline"} {
		r.POST(base+"/:id/"+a, authenticate, moderator, change(a))
	}
	r.POST(base+"/:id/payout-card", authenticate, func(c *gin.Context) {
		v, err := service.BindPayoutCard(c.Request.Context(), &wp.OwnedID{Id: c.Param("id"), UserId: c.GetString("user_id")})
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, gin.H{"url": v.Url})
	})
	r.POST("/api/v1/payments/payout", func(c *gin.Context) {
		_, err := service.PayoutCallback(c.Request.Context(), &wp.CardNotification{Xml: c.PostForm("pg_xml")})
		if err != nil {
			fail(c, err)
			return
		}
		c.Status(200)
	})
	r.POST("/api/v1/payments/card", func(c *gin.Context) {
		raw := c.PostForm("pg_xml")
		if raw == "" {
			c.Status(400)
			return
		}
		_, err := service.CardCallback(c.Request.Context(), &wp.CardNotification{Xml: raw})
		if err != nil {
			fail(c, err)
			return
		}
		c.Status(200)
	})
	r.POST(base+"/:id/checkout", func(c *gin.Context) {
		var body struct{ Amount int32 }
		if c.ShouldBindJSON(&body) != nil {
			c.Status(400)
			return
		}
		v, err := service.Checkout(c.Request.Context(), &wp.CheckoutRequest{AppealId: c.Param("id"), Amount: body.Amount})
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, gin.H{"url": v.Url})
	})
	r.POST("/api/v1/payments/result", func(c *gin.Context) {
		if err := c.Request.ParseForm(); err != nil {
			c.Status(400)
			return
		}
		fields := map[string]string{}
		for k, v := range c.Request.PostForm {
			if len(v) != 1 {
				c.Status(400)
				return
			}
			fields[k] = v[0]
		}
		ack, err := service.PaymentCallback(c.Request.Context(), &wp.PaymentResult{Fields: fields})
		if err != nil {
			fail(c, err)
			return
		}
		c.Data(200, "application/xml", []byte(ack.Xml))
	})
}
