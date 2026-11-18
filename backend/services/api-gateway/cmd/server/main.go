package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	hp "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"saya/api-gateway/internal/bootstrap"
	pb "saya/contracts/auth/v1"
	wp "saya/contracts/workflow/v1"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("gateway stopped", "error", e)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		return err
	}
	public, e := base64.RawURLEncoding.DecodeString(cfg.JWTPublic)
	if e != nil || len(public) != 32 {
		return errors.New("JWT_PUBLIC must encode 32 bytes")
	}
	keys := map[string]string{cfg.JWTKid: base64.RawURLEncoding.EncodeToString(public)}
	if raw := cfg.JWTPublicKeys; raw != "" {
		if e = json.Unmarshal([]byte(raw), &keys); e != nil {
			return errors.New("invalid JWT_PUBLIC_KEYS")
		}
	}
	publicKeys := map[string]ed25519.PublicKey{}
	jwks := []gin.H{}
	for kid, encoded := range keys {
		key, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil || len(key) != 32 || kid == "" {
			return errors.New("invalid public key")
		}
		publicKeys[kid] = key
		jwks = append(jwks, gin.H{"kty": "OKP", "crv": "Ed25519", "alg": "EdDSA", "use": "sig", "kid": kid, "x": encoded})
	}
	if len(keys) == 0 {
		return errors.New("empty key set")
	}
	conn, e := grpc.NewClient(cfg.AuthAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		return e
	}
	defer conn.Close()
	client := pb.NewAuthClient(conn)
	surveyConn, e := grpc.NewClient(cfg.SurveyAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		return e
	}
	defer surveyConn.Close()
	survey := wp.NewSurveyClient(surveyConn)
	assessmentConn, e := grpc.NewClient(cfg.AssessmentAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		return e
	}
	defer assessmentConn.Close()
	assessment := wp.NewAssessmentClient(assessmentConn)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() >= 400 && c.Writer.Size() == -1 {
			c.JSON(c.Writer.Status(), gin.H{"error": gin.H{"code": http.StatusText(c.Writer.Status()), "request_id": c.Writer.Header().Get("X-Request-ID")}})
		}
	})
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "saya_http_requests_total", Help: "Requests by route and status"}, []string{"route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "saya_http_request_duration_seconds", Help: "Request latency", Buckets: prometheus.DefBuckets}, []string{"route"})
	registry.MustRegister(requests, duration)
	for name, address := range map[string]string{"auth": cfg.AuthAddress, "survey": cfg.SurveyAddress, "assessment": cfg.AssessmentAddress, "ai": cfg.AIAddress} {
		name, address := name, address
		connection, e := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if e != nil {
			return e
		}
		defer connection.Close()
		registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "saya_dependency_up", Help: "Internal service health", ConstLabels: prometheus.Labels{"service": name}}, func() float64 {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			v, e := hp.NewHealthClient(connection).Check(ctx, &hp.HealthCheckRequest{})
			if e == nil && v.Status == hp.HealthCheckResponse_SERVING {
				return 1
			}
			return 0
		}))
	}

	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		requests.WithLabelValues(route, strconv.Itoa(c.Writer.Status())).Inc()
		duration.WithLabelValues(route).Observe(time.Since(start).Seconds())
	})
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
	r.SetTrustedProxies(nil)
	var mu sync.Mutex
	counts := map[string]int{}
	window := time.Now()
	r.Use(func(c *gin.Context) {
		id := uuid.NewString()
		c.Header("X-Request-ID", id)
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		limit := int64(16384)
		if c.Request.URL.Path == "/api/v1/surveys/submissions" {
			limit = 96 << 10
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		origin := c.GetHeader("Origin")
		if origin != "" {
			if origin != cfg.WebOrigin {
				c.AbortWithStatus(403)
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		mu.Lock()
		if time.Since(window) > time.Minute {
			counts = map[string]int{}
			window = time.Now()
		}
		key := c.ClientIP()
		counts[key]++
		limited := counts[key] > 120 || len(counts) > 10000
		if c.FullPath() == "/metrics" || strings.HasPrefix(c.FullPath(), "/health/") {
			limited = false
		}
		mu.Unlock()
		if limited {
			c.AbortWithStatusJSON(429, gin.H{"error": gin.H{"code": "rate_limited", "request_id": id}})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	fail := func(c *gin.Context, e error) {
		code := 500
		switch status.Code(e) {
		case codes.InvalidArgument:
			code = 400
		case codes.ResourceExhausted:
			code = 429
		case codes.AlreadyExists:
			code = 409
		case codes.Unauthenticated:
			code = 401
		case codes.NotFound:
			code = 404
		case codes.Unavailable, codes.DeadlineExceeded:
			code = 503
		}
		c.JSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "request_id": c.Writer.Header().Get("X-Request-ID")}})
	}
	cookie := func(c *gin.Context, name, value string, age int, httpOnly bool) {
		http.SetCookie(c.Writer, &http.Cookie{Name: name, Value: value, Path: "/api/v1/auth", MaxAge: age, HttpOnly: httpOnly, Secure: cfg.CookieSecure, SameSite: http.SameSiteStrictMode})
	}
	send := func(c *gin.Context, v *pb.Session, code int) {
		if c.GetHeader("X-Saya-Client") == "native" && c.GetHeader("Origin") == "" {
			c.JSON(code, gin.H{"access_token": v.AccessToken, "refresh_token": v.RefreshToken, "user": gin.H{"id": v.User.Id, "email": v.User.Email, "role": v.User.Role}})
			return
		}
		cookie(c, "saya_refresh", v.RefreshToken, 2592000, true)
		b := make([]byte, 32)
		if _, e := rand.Read(b); e != nil {
			c.Status(500)
			return
		}
		cookie(c, "saya_csrf", base64.RawURLEncoding.EncodeToString(b), 2592000, false)
		c.JSON(code, gin.H{"access_token": v.AccessToken, "csrf_token": base64.RawURLEncoding.EncodeToString(b), "user": gin.H{"id": v.User.Id, "email": v.User.Email, "role": v.User.Role}})
	}
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var body pb.RegistrationInput
		if c.ShouldBindJSON(&body) != nil {
			c.Status(400)
			return
		}
		v, err := client.StartRegistration(c.Request.Context(), &body)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(202, gin.H{"registration_id": v.Id})
	})
	r.POST("/api/v1/auth/register/confirm", func(c *gin.Context) {
		var body struct {
			ID   string `json:"registration_id"`
			Code string `json:"code"`
		}
		if c.ShouldBindJSON(&body) != nil {
			c.Status(400)
			return
		}
		v, err := client.ConfirmRegistration(c.Request.Context(), &pb.RegistrationCode{Id: body.ID, Code: body.Code})
		if err != nil {
			fail(c, err)
			return
		}
		send(c, v, 201)
	})
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var q pb.Credentials
		if c.ShouldBindJSON(&q) != nil {
			c.Status(400)
			return
		}
		v, err := client.Login(c.Request.Context(), &q)
		if err != nil {
			fail(c, err)
			return
		}
		send(c, v, 200)
	})
	r.GET("/api/v1/auth/csrf", func(c *gin.Context) {
		csrf, e := c.Cookie("saya_csrf")
		if e != nil || csrf == "" {
			c.Status(401)
			return
		}
		c.JSON(200, gin.H{"csrf_token": csrf})
	})
	for _, action := range []string{"refresh", "logout"} {
		action := action
		r.POST("/api/v1/auth/"+action, func(c *gin.Context) {
			raw, e := c.Cookie("saya_refresh")
			if c.GetHeader("X-Saya-Client") == "native" && c.GetHeader("Origin") == "" {
				var body struct {
					Token string `json:"refresh_token"`
				}
				if c.ShouldBindJSON(&body) != nil || len(body.Token) != 43 {
					c.Status(400)
					return
				}
				raw = body.Token
				e = nil
			} else {
				csrf, f := c.Cookie("saya_csrf")
				if e != nil || f != nil || csrf == "" || c.GetHeader("X-CSRF-Token") != csrf {
					c.Status(403)
					return
				}
			}
			q := &pb.RefreshRequest{Token: raw}
			if action == "logout" {
				_, e = client.Logout(c.Request.Context(), q)
				cookie(c, "saya_refresh", "", -1, true)
				cookie(c, "saya_csrf", "", -1, false)
				if e != nil {
					fail(c, e)
					return
				}
				c.Status(204)
				return
			}
			v, e := client.Refresh(c.Request.Context(), q)
			if e != nil {
				fail(c, e)
				return
			}
			send(c, v, 200)
		})
	}
	authenticate := func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		t, e := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			kid, _ := t.Header["kid"].(string)
			key, ok := publicKeys[kid]
			if !ok {
				return nil, errors.New("unknown key")
			}
			return key, nil
		}, jwt.WithValidMethods([]string{"EdDSA"}), jwt.WithIssuer("saya-auth"), jwt.WithAudience("saya-api"), jwt.WithExpirationRequired())
		if e != nil || !t.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"code": "unauthorized", "request_id": c.Writer.Header().Get("X-Request-ID")}})
			return
		}
		id, e := t.Claims.GetSubject()
		if e != nil || id == "" {
			c.AbortWithStatus(401)
			return
		}
		c.Set("user_id", id)
		c.Next()
	}

	aiConn, err := grpc.NewClient(cfg.AIAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer aiConn.Close()
	interviewer := wp.NewInterviewAssistantClient(aiConn)
	r.POST("/api/v1/surveys/interview/focus", authenticate, func(c *gin.Context) {
		var body wp.FocusRequest
		if c.ShouldBindJSON(&body) != nil {
			c.Status(400)
			return
		}
		v, err := interviewer.Focus(c.Request.Context(), &body)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, gin.H{"focus": v.Focus})
	})
	r.GET("/api/v1/surveys/latest", authenticate, func(c *gin.Context) {
		op, err := survey.Latest(c.Request.Context(), &wp.Owner{UserId: c.GetString("user_id")})
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, gin.H{"id": op.Id})
	})
	for path, method := range map[string]func(context.Context, *wp.Empty, ...grpc.CallOption) (*wp.Questionnaire, error){"/api/v1/surveys/current": survey.Current, "/api/v1/surveys/interview": survey.Interview} {
		r.GET(path, authenticate, func(c *gin.Context) {
			v, e := method(c.Request.Context(), &wp.Empty{})
			if e != nil {
				fail(c, e)
				return
			}
			questions := []gin.H{}
			for _, q := range v.Questions {
				questions = append(questions, gin.H{"id": q.Id, "label": q.Label, "variants": q.Variants})
			}
			c.JSON(200, gin.H{"id": v.Id, "version": v.Version, "questions": questions, "disclaimer": v.Disclaimer})
		})
	}
	r.POST("/api/v1/surveys/submissions", authenticate, func(c *gin.Context) {
		var body struct {
			Responses       map[string]string `json:"responses"`
			Ratings         map[string]int32  `json:"ratings"`
			Summary         string            `json:"summary"`
			Needs           []string          `json:"needs"`
			QuestionFocus   map[string]string `json:"question_focus"`
			QuestionnaireID string            `json:"questionnaire_id"`
			Version         int32             `json:"version"`
			Answers         map[string]bool   `json:"answers"`
			Consent         bool              `json:"consent"`
			ExternalConsent bool              `json:"external_llm_consent"`
			IdempotencyKey  string            `json:"idempotency_key"`
		}
		if c.ShouldBindJSON(&body) != nil {
			c.Status(400)
			return
		}
		v, e := survey.Submit(c.Request.Context(), &wp.Submission{UserId: c.GetString("user_id"), QuestionnaireId: body.QuestionnaireID, Version: body.Version, QuestionFocus: body.QuestionFocus, Responses: body.Responses, Ratings: body.Ratings, Summary: body.Summary, Needs: body.Needs, Answers: body.Answers, Consent: body.Consent, ExternalLlmConsent: body.ExternalConsent, IdempotencyKey: body.IdempotencyKey})
		if e != nil {
			fail(c, e)
			return
		}
		c.JSON(202, gin.H{"id": v.Id, "status": v.Status})
	})
	r.GET("/api/v1/weekly/history", authenticate, func(c *gin.Context) {
		v, err := assessment.History(c.Request.Context(), &wp.HistoryRequest{UserId: c.GetString("user_id"), BeforeId: c.Query("before")})
		if err != nil {
			fail(c, err)
			return
		}
		entries := make([]gin.H, 0, len(v.Entries))
		for _, e := range v.Entries {
			entries = append(entries, gin.H{"id": e.Id, "completed_at": e.CompletedAt, "version": e.Version, "percentage": e.Percentage, "ratings": e.Ratings, "directions": e.Directions})
		}
		c.JSON(200, gin.H{"entries": entries, "next_due": v.NextDue, "reminder_due": v.ReminderDue, "next_cursor": v.NextCursor, "push_available": v.PushAvailable})
	})
	r.POST("/api/v1/weekly/device", authenticate, func(c *gin.Context) {
		var body struct {
			Token   string `json:"installation_id"`
			Enabled *bool  `json:"enabled"`
		}
		if c.ShouldBindJSON(&body) != nil || body.Enabled == nil {
			c.Status(400)
			return
		}
		_, err := assessment.SetDevice(c.Request.Context(), &wp.DeviceRequest{UserId: c.GetString("user_id"), Token: body.Token, Enabled: *body.Enabled})
		if err != nil {
			fail(c, err)
			return
		}
		c.Status(204)
	})
	r.GET("/api/v1/assessments/:id", authenticate, func(c *gin.Context) {
		if _, e := uuid.Parse(c.Param("id")); e != nil {
			c.Status(400)
			return
		}
		v, e := assessment.Get(c.Request.Context(), &wp.OwnedID{Id: c.Param("id"), UserId: c.GetString("user_id")})
		if e != nil {
			fail(c, e)
			return
		}
		c.JSON(200, gin.H{"id": v.Id, "status": v.Status, "version": v.Version, "summary": v.Summary, "ratings": v.Ratings, "percentage": v.Percentage, "directions": v.Directions, "immediate_threat_reported": v.ImmediateThreatReported, "recommendations": v.Recommendations, "disclaimer": v.Disclaimer, "provider": v.Provider})
	})
	r.POST("/api/v1/support/vacancies", authenticate, vacancySearch(&http.Client{Timeout: 6 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, cfg.HHAppToken))
	r.GET("/api/v1/users/me", authenticate, func(c *gin.Context) {
		u, e := client.Me(c.Request.Context(), &pb.Identity{UserId: c.GetString("user_id")})
		if e != nil {
			fail(c, e)
			return
		}
		c.JSON(200, gin.H{"id": u.Id, "email": u.Email, "role": u.Role})
	})
	for _, kind := range []string{"connection", "subscription"} {
		kind := kind
		r.POST("/api/v1/realtime/"+kind+"-token", authenticate, func(c *gin.Context) {
			channel := ""
			if kind == "subscription" {
				var body struct {
					Channel string `json:"channel"`
				}
				if c.ShouldBindJSON(&body) != nil || body.Channel == "" {
					c.Status(400)
					return
				}
				channel = body.Channel
			}
			v, e := client.Realtime(c.Request.Context(), &pb.RealtimeRequest{UserId: c.GetString("user_id"), Channel: channel})
			if e != nil {
				fail(c, e)
				return
			}
			c.JSON(200, gin.H{"token": v.Token, "expires_at": v.ExpiresAt, "channel": v.Channel})
		})
	}
	r.GET("/health/ready", func(c *gin.Context) {
		v, e := hp.NewHealthClient(conn).Check(c.Request.Context(), &hp.HealthCheckRequest{})
		if e != nil || v.Status != hp.HealthCheckResponse_SERVING {
			c.Status(503)
			return
		}
		c.Status(200)
	})
	r.GET("/.well-known/jwks.json", func(c *gin.Context) {
		c.JSON(200, gin.H{"keys": jwks})
	})
	r.GET("/health/live", func(c *gin.Context) { c.Status(200) })
	server := &http.Server{Addr: ":8080", Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if e := server.Shutdown(shutdown); e != nil {
			slog.Error("shutdown failed")
		}
	}()
	e = server.ListenAndServe()
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}
