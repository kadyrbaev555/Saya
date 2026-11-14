package main

import (
	"context"
	"google.golang.org/grpc"
	"log/slog"
	"os"
	pb "saya/contracts/workflow/v1"
	"saya/messaging"
	"saya/platform"
	"saya/survey-service/internal/adapters"
	"saya/survey-service/internal/application"
	"saya/survey-service/internal/bootstrap"
)

func main() {
	if e := run(); e != nil {
		slog.Error("survey stopped")
		os.Exit(1)
	}
}
func run() error {
	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		return err
	}
	ctx, cancel := platform.Context()
	defer cancel()
	db, e := platform.DB(ctx, cfg.DatabaseURL)
	if e != nil {
		return e
	}
	defer db.Close()
	bus, e := messaging.Open(cfg.RabbitURL, "saya.assessment.v1")
	if e != nil {
		return e
	}
	defer bus.Close()
	platform.Loop(ctx, func(c context.Context) error {
		e := platform.Deliver(c, db, bus.Publish)
		if e != nil && bus.IsClosed() {
			cancel()
		}
		return e
	})
	return platform.Serve(ctx, func(s *grpc.Server) {
		pb.RegisterSurveyServer(s, &adapters.Server{Service: application.Service{Repo: adapters.Repository{DB: db}}})
	})
}
