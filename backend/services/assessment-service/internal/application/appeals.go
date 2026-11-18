package application

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrAppealUnavailable = errors.New("appeal unavailable")
var ErrInvalidAppeal = errors.New("invalid appeal")

type Appeal struct {
	ID, Description, State string
	Reported, CardReady    bool
}
type Appeals interface {
	ListAppeals(context.Context, string, string) ([]Appeal, error)
	SubmitAppeal(context.Context, string, string, string) error
	ChangeAppeal(context.Context, string, string, string) error
}
type AppealService struct{ Store Appeals }

func (s AppealService) Submit(ctx context.Context, user, assessment, description string, consent bool) error {
	description = strings.TrimSpace(description)
	if _, err := uuid.Parse(assessment); err != nil || !consent || description == "" || utf8.RuneCountInString(description) > 1200 {
		return ErrInvalidAppeal
	}
	return s.Store.SubmitAppeal(ctx, user, assessment, description)
}
