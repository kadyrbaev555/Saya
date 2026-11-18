package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"saya/ai-service/internal/application"
	pb "saya/contracts/workflow/v1"
)

type InterviewServer struct {
	pb.UnimplementedInterviewAssistantServer
	Assistant application.InterviewAssistant
}

func (s InterviewServer) Focus(ctx context.Context, r *pb.FocusRequest) (*pb.FocusResult, error) {
	v, e := s.Assistant.Focus(ctx, r.Topic, r.Ratings, r.Consent)
	if errors.Is(e, application.ErrFocusInput) {
		return nil, status.Error(codes.InvalidArgument, "invalid focus request")
	}
	if e != nil {
		return nil, status.Error(codes.Unavailable, "assistant unavailable")
	}
	return &pb.FocusResult{Focus: v}, nil
}
func (p Gemini) Focus(ctx context.Context, topic string, ratings map[string]int32) (string, error) {
	data, err := json.Marshal(map[string]any{"topic": topic, "self_ratings": ratings})
	if err != nil {
		return "", err
	}
	schema := map[string]any{"type": "object", "required": []string{"focus"}, "additionalProperties": false, "properties": map[string]any{"focus": map[string]any{"type": "string", "enum": []string{"routine", "resources", "obstacles"}}}}
	var answer struct {
		Focus string `json:"focus"`
	}
	err = p.structured(ctx, "Choose the perspective of the next approved open-ended question for this topic. Consider the weekly self-ratings (0 means needs were unmet, 4 means fully met). Choose resources for exploring support, obstacles for exploring difficulties, routine for a general check-in. No diagnosis or risk scoring.", data, schema, &answer)
	return answer.Focus, err
}
