package application

import (
	"context"
	"net/mail"
	"saya/auth-service/internal/domain"
	"strings"
)

type Repository interface {
	Create(context.Context, string, string) (domain.User, error)
	Find(context.Context, string) (domain.User, error)
	Get(context.Context, string) (domain.User, error)
	NewSession(context.Context, string, string) error
	Rotate(context.Context, string, string) (domain.User, error)
	Revoke(context.Context, string) error
}
type Password interface {
	Hash(string) (string, error)
	Verify(string, string) bool
}
type Tokens interface {
	Access(domain.User) (string, error)
	Opaque() (string, string, error)
	Digest(string) string
}
type Service struct {
	Repo     Repository
	Password Password
	Tokens   Tokens
}

func (s *Service) Register(ctx context.Context, email, password string) (domain.Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	a, e := mail.ParseAddress(email)
	if e != nil || a.Address != email || len(email) > 254 || len(password) < 12 || len(password) > 128 {
		return domain.Session{}, domain.ErrValidation
	}
	h, e := s.Password.Hash(password)
	if e != nil {
		return domain.Session{}, e
	}
	u, e := s.Repo.Create(ctx, email, h)
	if e != nil {
		return domain.Session{}, e
	}
	return s.session(ctx, u)
}
func (s *Service) Login(ctx context.Context, email, password string) (domain.Session, error) {
	if len(password) > 128 {
		return domain.Session{}, domain.ErrCredentials
	}
	u, e := s.Repo.Find(ctx, strings.ToLower(strings.TrimSpace(email)))
	if e != nil {
		s.Password.Verify(password, "")
		return domain.Session{}, domain.ErrCredentials
	}
	if !s.Password.Verify(password, u.Password) {
		return domain.Session{}, domain.ErrCredentials
	}
	return s.session(ctx, u)
}
func (s *Service) session(ctx context.Context, u domain.User) (domain.Session, error) {
	raw, h, e := s.Tokens.Opaque()
	if e != nil {
		return domain.Session{}, e
	}
	a, e := s.Tokens.Access(u)
	if e != nil {
		return domain.Session{}, e
	}
	if e = s.Repo.NewSession(ctx, u.ID, h); e != nil {
		return domain.Session{}, e
	}
	return domain.Session{Access: a, Refresh: raw, User: u}, nil
}
func (s *Service) Refresh(ctx context.Context, raw string) (domain.Session, error) {
	if len(raw) != 43 {
		return domain.Session{}, domain.ErrCredentials
	}
	next, h, e := s.Tokens.Opaque()
	if e != nil {
		return domain.Session{}, e
	}
	u, e := s.Repo.Rotate(ctx, s.Tokens.Digest(raw), h)
	if e != nil {
		return domain.Session{}, e
	}
	a, e := s.Tokens.Access(u)
	return domain.Session{Access: a, Refresh: next, User: u}, e
}
func (s *Service) Logout(ctx context.Context, raw string) error {
	return s.Repo.Revoke(ctx, s.Tokens.Digest(raw))
}
