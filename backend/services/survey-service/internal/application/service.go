package application

import (
	"context"
	"github.com/google/uuid"
	"saya/survey-service/internal/domain"
)

type Repository interface {
	Save(context.Context, domain.Submission) (string, error)
	Input(context.Context, string, string) (domain.Submission, error)
}
type Service struct{ Repo Repository }

func (s Service) Submit(ctx context.Context, v domain.Submission) (string, error) {
	if _, e := uuid.Parse(v.UserID); e != nil {
		return "", domain.ErrInvalid
	}
	if _, e := uuid.Parse(v.IdempotencyKey); e != nil {
		return "", domain.ErrInvalid
	}
	if v.QuestionnaireID != "saya-demo" || v.Version != 1 || !v.Consent || len(v.Answers) != 2 {
		return "", domain.ErrInvalid
	}
	for _, key := range []string{"immediate_threat", "information_requested"} {
		if _, ok := v.Answers[key]; !ok {
			return "", domain.ErrInvalid
		}
	}
	return s.Repo.Save(ctx, v)
}
