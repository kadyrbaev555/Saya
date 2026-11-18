package application

import (
	"context"
	"saya/assessment-service/internal/domain"
)

type Repository interface {
 History(context.Context,string,string)(domain.WeeklyHistory,error)
 SetDevice(context.Context,string,string,bool)error
	Create(context.Context, string, string, domain.Signal) error
	Input(context.Context, string, string) (domain.Signal, error)
	Get(context.Context, string, string) (domain.Result, error)
	Complete(context.Context, string, string, []string, string, bool) error
}
type InputSource interface {
	Fetch(context.Context, string, string) (domain.Signal, error)
}
type Service struct {
	Repo   Repository
	Source InputSource
}

func (s Service) Start(ctx context.Context, id, user string) error {
	signal, e := s.Source.Fetch(ctx, id, user)
	if e != nil {
		return e
	}
	return s.Repo.Create(ctx, id, user, signal)
}
func (s Service) Get(ctx context.Context, id, user string) (domain.Result, error) {
	result, e := s.Repo.Get(ctx, id, user)
	if e == domain.ErrNotFound {
		signal, err := s.Source.Fetch(ctx, id, user)
		if err != nil {
			return result, err
		}
		return domain.Build(id, "pending", signal, nil, ""), nil
	}
	return result, e
}
func (s Service) Complete(ctx context.Context, id, user string, codes []string, provider string, failed bool) error {
	if provider == "self-report" && !failed && len(codes) == 0 {
		signal, err := s.Repo.Input(ctx, id, user)
		if err != nil {
			return err
		}
		if signal.SourceVersion != 2 {
			return domain.ErrInvalid
		}
		return s.Repo.Complete(ctx, id, user, nil, provider, false)
	}
	if !failed {
		if e := domain.Validate(codes); e != nil {
			return e
		}
		if provider != "demonstration-mock" && provider != "external-http" && provider != "gemini" {
			return domain.ErrInvalid
		}
	}
	return s.Repo.Complete(ctx, id, user, codes, provider, failed)
}
