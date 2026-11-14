package application

import (
	"context"
	"saya/auth-service/internal/domain"
)

type RealtimeSigner interface {
	Sign(string, string) (string, int64, error)
}

func (s *Service) Realtime(ctx context.Context, user, channel string, signer RealtimeSigner) (string, int64, error) {
	if _, e := s.Repo.Get(ctx, user); e != nil {
		return "", 0, domain.ErrCredentials
	}
	if channel != "" && channel != "user:"+user+":assessments" {
		return "", 0, domain.ErrCredentials
	}
	return signer.Sign(user, channel)
}
