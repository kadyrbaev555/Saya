package application

import (
	"context"
	"testing"
)

type focusStub struct {
	calls int
	value string
}

func (f *focusStub) Focus(context.Context, string, map[string]int32) (string, error) {
	f.calls++
	return f.value, nil
}
func TestInterviewFocusConsentAndValidation(t *testing.T) {
	p := &focusStub{value: "resources"}
	s := InterviewAssistant{Provider: p}
	for _, tc := range []struct {
		topic   string
		ratings map[string]int32
		consent bool
	}{{"work", nil, false}, {"invalid", nil, true}, {"work", map[string]int32{"work": 5}, true}} {
		if _, err := s.Focus(context.Background(), tc.topic, tc.ratings, tc.consent); err != ErrFocusInput {
			t.Fatal("invalid input accepted")
		}
	}
	if p.calls != 0 {
		t.Fatal("unexpected external call")
	}
	if f, e := s.Focus(context.Background(), "work", nil, true); e != nil || f != "resources" {
		t.Fatal("expected approved focus")
	}
	p.value = "other"
	if _, e := s.Focus(context.Background(), "work", nil, true); e != ErrFocusUnavailable {
		t.Fatal("unapproved focus accepted")
	}
}
