package adapters

import (
	"context"
	"saya/ai-service/internal/domain"
	pb "saya/contracts/workflow/v1"
)

type Assessment struct{ Client pb.AssessmentClient }

func (a Assessment) Input(ctx context.Context, id, user string) (domain.Input, error) {
	v, e := a.Client.Input(ctx, &pb.OwnedID{Id: id, UserId: user})
	if e != nil {
		return domain.Input{}, e
	}
	return domain.Input{ImmediateThreat: v.Answers["immediate_threat"], InformationRequested: v.Answers["information_requested"], ExternalConsent: v.ExternalLlmConsent}, nil
}
func (a Assessment) Complete(ctx context.Context, id, user string, v domain.Output, provider string, failed bool) error {
	_, e := a.Client.Complete(ctx, &pb.Completion{Id: id, UserId: user, Recommendations: v.Recommendations, Provider: provider, Failed: failed})
	return e
}
