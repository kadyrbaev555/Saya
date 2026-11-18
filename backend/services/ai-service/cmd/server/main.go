package main

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"os"
	"saya/ai-service/internal/adapters"
	"saya/ai-service/internal/application"
	"saya/ai-service/internal/bootstrap"
	pb "saya/contracts/workflow/v1"
	wp "saya/contracts/workflow/v1"
	"saya/messaging"
	"saya/platform"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("AI worker stopped")
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
	connection, e := grpc.NewClient(cfg.AssessmentAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		return e
	}
	defer connection.Close()
	var provider application.LLMProvider = adapters.Mock{}
	callTimeout := 3 * time.Second
	switch cfg.LLMProvider {
	case "", "mock":
	case "gemini":
		configured, err := adapters.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			return err
		}
		provider = configured
		callTimeout = 10 * time.Second
	case "http":
		if cfg.LLMAPIKey == "" || cfg.LLMEndpoint == "" {
			return errors.New("LLM configuration required")
		}
		provider = adapters.HTTPProvider{Endpoint: cfg.LLMEndpoint, APIKey: cfg.LLMAPIKey}
	default:
		return errors.New("unknown LLM_PROVIDER")
	}
	service := application.Service{Provider: provider, CallTimeout: callTimeout, Assessment: adapters.Assessment{Client: pb.NewAssessmentClient(connection)}}
	bus, e := messaging.Open(cfg.RabbitURL, "saya.ai.v1")
	if e != nil {
		return e
	}
	defer bus.Close()
	go func() {
		if e := bus.Consume(ctx, func(c context.Context, event messaging.Event) error {
			return service.Process(c, event.OperationID, event.UserID)
		}); e != nil {
			slog.Error("AI consumer unavailable")
			cancel()
		}
	}()
	assistant := application.InterviewAssistant{}
	if gemini, ok := provider.(adapters.Gemini); ok {
		assistant.Provider = gemini
	}
	return platform.Serve(ctx, func(server *grpc.Server) {
		wp.RegisterInterviewAssistantServer(server, adapters.InterviewServer{Assistant: assistant})
	})
}
