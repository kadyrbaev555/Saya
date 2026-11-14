package adapters

import (
	"context"
	"saya/ai-service/internal/domain"
)

type Mock struct{}

func (Mock) Name() string { return "demonstration-mock" }
func (Mock) Generate(ctx context.Context, input domain.Input) (domain.Output, error) {
	if e := ctx.Err(); e != nil {
		return domain.Output{}, e
	}
	codes := []string{"review_support_information"}
	if input.ImmediateThreat {
		codes = append(codes, "choose_safe_device", "consider_trusted_contact")
	}
	return domain.Output{Recommendations: codes}, nil
}
