package main

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	hp "google.golang.org/grpc/health/grpc_health_v1"
	"log/slog"
	"net"
	"os"
	"os/signal"
	in "saya/auth-service/internal/adapters/inbound/grpc"
	"saya/auth-service/internal/adapters/outbound/email"
	"saya/auth-service/internal/adapters/outbound/password"
	"saya/auth-service/internal/adapters/outbound/postgres"
	"saya/auth-service/internal/adapters/outbound/token"
	"saya/auth-service/internal/application"
	"saya/auth-service/internal/bootstrap"
	pb "saya/contracts/auth/v1"
	"syscall"
	"time"
)

func run() error {
	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	seed, e := base64.RawURLEncoding.DecodeString(cfg.JWTSeed)
	if e != nil || len(seed) != 32 {
		return fmtError("JWT_SEED must encode 32 bytes")
	}
	db, e := sqlx.Open("postgres", cfg.DatabaseURL)
	if e != nil {
		return e
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	ping, stop := context.WithTimeout(ctx, 5*time.Second)
	defer stop()
	if e = db.PingContext(ping); e != nil {
		return e
	}
	hasher, e := password.New(cfg.ArgonMemory, cfg.ArgonIterations, cfg.ArgonThreads)
	if e != nil {
		return e
	}
	service := &application.Service{Repo: postgres.Repository{DB: db}, Password: hasher, Tokens: token.Provider{Key: ed25519.NewKeyFromSeed(seed), Kid: cfg.JWTKid}}
	server := grpc.NewServer()
	secret := cfg.CentrifugoTokenSecret
	if len(secret) < 32 {
		return fmtError("CENTRIFUGO_TOKEN_SECRET must be at least 32 bytes")
	}
	key := hmac.New(sha256.New, seed)
	key.Write([]byte("saya/auth/registration-code/v1"))
	registration := application.Registration{
		Store: postgres.Repository{DB: db},
		Mailer: email.Sender{
			Address: cfg.SMTPAddress, From: cfg.SMTPFrom,
			Username: cfg.SMTPUser, Password: cfg.SMTPPassword,
		},
		Password: hasher, Tokens: service.Tokens, CodeKey: key.Sum(nil),
	}
	pb.RegisterAuthServer(server, &in.Server{Registration: registration, Service: service, RealtimeSigner: token.Realtime{Secret: []byte(secret)}})
	hs := health.NewServer()
	hp.RegisterHealthServer(server, hs)
	hs.SetServingStatus("", hp.HealthCheckResponse_SERVING)
	l, e := net.Listen("tcp", ":9000")
	if e != nil {
		return e
	}
	go func() {
		<-ctx.Done()
		hs.Shutdown()
		done := make(chan struct{})
		go func() { server.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			server.Stop()
		}
	}()
	slog.Info("auth listening")
	e = server.Serve(l)
	if errors.Is(e, grpc.ErrServerStopped) {
		return nil
	}
	return e
}

type fmtError string

func (e fmtError) Error() string { return string(e) }
func main() {
	if e := run(); e != nil {
		slog.Error("auth stopped")
		os.Exit(1)
	}
}
