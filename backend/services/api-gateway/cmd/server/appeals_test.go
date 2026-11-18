package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	auth "saya/contracts/auth/v1"
	pb "saya/contracts/workflow/v1"
)

type appealIdentity struct {
	auth.AuthClient
	role string
}

func (a *appealIdentity) Me(context.Context, *auth.Identity, ...grpc.CallOption) (*auth.User, error) {
	return &auth.User{Role: a.role}, nil
}

type appealRoutes struct {
	pb.AssessmentClient
	last *pb.AppealChange
}

func (a *appealRoutes) MutateAppeal(_ context.Context, v *pb.AppealChange, _ ...grpc.CallOption) (*pb.Empty, error) {
	a.last = v
	return &pb.Empty{}, nil
}
func TestAppealAuthorization(t *testing.T) {
	service := &appealRoutes{}
	identity := &appealIdentity{role: "USER"}
	router := gin.New()
	registerAppeals(router, service, identity, func(c *gin.Context) { c.Set("user_id", "authenticated-owner") }, func(c *gin.Context, err error) { t.Error(err); c.Status(500) })
	send := func(action string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/appeals/item/"+action, strings.NewReader(`{"description":"public","consent":true,"moderator":true,"user_id":"foreign"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}
	if send("approve") != 403 || service.last != nil {
		t.Fatal("user could moderate")
	}
	if send("submit") != 204 || service.last.UserId != "authenticated-owner" || service.last.Moderator || !service.last.Consent {
		t.Fatal("owner/consent forwarding")
	}
	identity.role = "ADMIN"
	if send("approve") != 204 || !service.last.Moderator {
		t.Fatal("admin denied")
	}
}
