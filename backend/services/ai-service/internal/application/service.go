package application

import (
	"context"
	"saya/ai-service/internal/domain"
	"time"
)

type LLMProvider interface {
	Generate(context.Context, domain.Input) (domain.Output, error)
	Name() string
}
type Assessment interface {
	Input(context.Context, string, string) (domain.Input, error)
	Complete(context.Context, string, string, domain.Output, string, bool) error
}
type Service struct {
	CallTimeout time.Duration
	Provider    LLMProvider
	Assessment  Assessment
}

func (s Service) Process(ctx context.Context, id, user string) error {
	input, e := s.Assessment.Input(ctx, id, user)
	if e != nil {
		return e
	}
	var output domain.Output
	if input.Version == 2 && (!input.ExternalConsent || s.Provider.Name() == "demonstration-mock") {
		return s.Assessment.Complete(ctx, id, user, output, "self-report", false)
	}
	if s.Provider.Name() != "demonstration-mock" && !input.ExternalConsent {
		return s.Assessment.Complete(ctx, id, user, output, "unavailable", true)
	}
	for attempt := 0; attempt < 3; attempt++ {
		timeout := s.CallTimeout
		if timeout <= 0 {
			timeout = 3 * time.Second
		}
		call, cancel := context.WithTimeout(ctx, timeout)
		output, e = s.Provider.Generate(call, input)
		cancel()
		if e == nil {
			e = output.Validate()
		}
		if e == nil {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt < 2 {
			timer := time.NewTimer(time.Duration(attempt+1) * 200 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	if input.Version == 2 && e != nil {
		return s.Assessment.Complete(ctx, id, user, domain.Output{}, "self-report", false)
	}
	return s.Assessment.Complete(ctx, id, user, output, s.Provider.Name(), e != nil)
}
