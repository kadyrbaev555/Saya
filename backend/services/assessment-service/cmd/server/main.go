package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"os"
	"saya/assessment-service/internal/adapters"
	"saya/assessment-service/internal/application"
	"saya/assessment-service/internal/bootstrap"
	pb "saya/contracts/workflow/v1"
	"saya/messaging"
	"saya/platform"
)

func main() {
	if e := run(); e != nil {
		slog.Error("assessment stopped")
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
	source, e := grpc.NewClient(cfg.SurveyAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		return e
	}
	defer source.Close()
	repo := adapters.Repository{DB: db}
	service := application.Service{Repo: repo, Source: adapters.Source{Client: pb.NewSurveyClient(source)}}
	publish, e := messaging.Open(cfg.RabbitURL, "saya.ai.v1")
	if e != nil {
		return e
	}
	defer publish.Close()
	consume, e := messaging.Open(cfg.RabbitURL, "saya.assessment.v1")
	if e != nil {
		return e
	}
	defer consume.Close()
	platform.Loop(ctx, func(c context.Context) error {
		e := platform.Deliver(c, db, publish.Publish)
		if e != nil && publish.IsClosed() {
			cancel()
		}
		return e
	})
	platform.Loop(ctx, func(c context.Context) error {
		return repo.Notify(c, cfg.CentrifugoURL, cfg.CentrifugoAPIKey)
	})
	platform.Loop(ctx, repo.AnnounceDue)
	var push *adapters.FCM
	if cfg.FCMCredentials != "" {
		push, e = adapters.NewFCM(ctx, cfg.FCMCredentials)
		if e != nil {
			return e
		}
		platform.Loop(ctx, func(c context.Context) error { return repo.SendDue(c, push) })
	}
	go func() {
		if e := consume.Consume(ctx, func(c context.Context, event messaging.Event) error {
			return service.Start(c, event.OperationID, event.UserID)
		}); e != nil {
			slog.Error("assessment consumer unavailable")
			cancel()
		}
	}()
	return platform.Serve(ctx, func(s *grpc.Server) {
		pb.RegisterAssessmentServer(s, &adapters.Server{Service: service, PushAvailable: push != nil})
	})
}
