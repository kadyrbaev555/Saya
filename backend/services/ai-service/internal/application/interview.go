package application

import (
	"context"
	"errors"
	"time"
)

type FocusProvider interface {
	Focus(context.Context, string, map[string]int32) (string, error)
}
type InterviewAssistant struct{ Provider FocusProvider }

var ErrFocusInput = errors.New("invalid interview focus request")
var ErrFocusUnavailable = errors.New("interview assistant unavailable")

func (a InterviewAssistant) Focus(ctx context.Context, topic string, ratings map[string]int32, consent bool) (string, error) {
	topics := map[string]bool{"housing": true, "expenses": true, "work": true, "connection": true, "daily": true}
	if !consent || (!topics[topic] && topic != "priority") {
		return "", ErrFocusInput
	}
	for k, n := range ratings {
		if !topics[k] || n < 0 || n > 4 {
			return "", ErrFocusInput
		}
	}
	if a.Provider == nil {
		return "", ErrFocusUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	focus, err := a.Provider.Focus(ctx, topic, ratings)
	if err != nil || (focus != "resources" && focus != "obstacles" && focus != "routine") {
		return "", ErrFocusUnavailable
	}
	return focus, nil
}
