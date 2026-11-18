package application

import (
	"context"
	"errors"
	"saya/ai-service/internal/domain"
	"testing"
)

type failingProvider struct{ calls int }

func (p *failingProvider) Name() string { return "external-http" }
func (p *failingProvider) Generate(context.Context, domain.Input) (domain.Output, error) {
	p.calls++
	return domain.Output{}, errors.New("injected timeout")
}

type assessmentStub struct {
	version         int32
	provider        string
	consent, failed bool
	completed       int
}

func (a *assessmentStub) Input(context.Context, string, string) (domain.Input, error) {
	return domain.Input{Version: a.version, ExternalConsent: a.consent}, nil
}
func (a *assessmentStub) Complete(ctx context.Context, id, user string, output domain.Output, provider string, failed bool) error {
	a.provider = provider
	a.failed = failed
	a.completed++
	return nil
}
func TestConsentAndBoundedRetry(t *testing.T) {
	for _, consent := range []bool{false, true} {
		provider := &failingProvider{}
		assessment := &assessmentStub{consent: consent}
		if e := (Service{Provider: provider, Assessment: assessment}).Process(context.Background(), "job", "user"); e != nil {
			t.Fatal(e)
		}
		expected := 0
		if consent {
			expected = 3
		}
		if provider.calls != expected || assessment.completed != 1 || !assessment.failed {
			t.Fatalf("consent=%v calls=%d completion=%d failed=%v", consent, provider.calls, assessment.completed, assessment.failed)
		}
	}
}

func TestInterviewLocalCompletion(t *testing.T) {
	for _, consent := range []bool{false, true} {
		p := &failingProvider{}
		a := &assessmentStub{version: 2, consent: consent}
		if err := (Service{Assessment: a, Provider: p}).Process(context.Background(), "id", "user"); err != nil {
			t.Fatal(err)
		}
		want := 0
		if consent {
			want = 3
		}
		if p.calls != want || a.failed || a.provider != "self-report" || a.completed != 1 {
			t.Fatal("incorrect consent/fallback behaviour")
		}
	}
}
