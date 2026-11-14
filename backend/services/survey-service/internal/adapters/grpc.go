package adapters

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "saya/contracts/workflow/v1"
	"saya/survey-service/internal/application"
	"saya/survey-service/internal/domain"
)

type Server struct {
	pb.UnimplementedSurveyServer
	Service application.Service
}

func (s Server) Current(context.Context, *pb.Empty) (*pb.Questionnaire, error) {
	return &pb.Questionnaire{Id: "saya-demo", Version: 1, Disclaimer: "Демонстрационный добровольный опрос; не валидированная оценка риска. Не вводите реальные кризисные данные.", Questions: []*pb.Question{{Id: "immediate_threat", Label: "Демонстрация: сообщена непосредственная угроза?"}, {Id: "information_requested", Label: "Демонстрация: нужна информация о поддержке?"}}}, nil
}
func (s Server) Submit(ctx context.Context, v *pb.Submission) (*pb.Operation, error) {
	id, e := s.Service.Submit(ctx, domain.Submission{UserID: v.UserId, QuestionnaireID: v.QuestionnaireId, Version: v.Version, Answers: v.Answers, Consent: v.Consent, ExternalConsent: v.ExternalLlmConsent, IdempotencyKey: v.IdempotencyKey})
	if errors.Is(e, domain.ErrInvalid) {
		return nil, status.Error(codes.InvalidArgument, "invalid demonstration submission")
	}
	if e != nil {
		return nil, status.Error(codes.Internal, "submission failed")
	}
	return &pb.Operation{Id: id, Status: "pending"}, nil
}
func (s Server) Input(ctx context.Context, v *pb.OwnedID) (*pb.Submission, error) {
	data, e := s.Service.Repo.Input(ctx, v.Id, v.UserId)
	if e != nil {
		return nil, status.Error(codes.NotFound, "submission unavailable")
	}
	return &pb.Submission{UserId: data.UserID, QuestionnaireId: data.QuestionnaireID, Version: data.Version, Answers: data.Answers, Consent: data.Consent, ExternalLlmConsent: data.ExternalConsent, IdempotencyKey: data.IdempotencyKey}, nil
}
