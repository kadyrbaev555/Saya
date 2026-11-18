package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"saya/auth-service/internal/domain"
)

var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var codePattern = regexp.MustCompile(`^[0-9]{6}$`)
var ErrDelivery = errors.New("registration email unavailable")
var ErrResendTooSoon = errors.New("registration request too recent")

type PendingRegistration struct {
	ID, Email, FullName, Phone, PasswordHash string
	CodeDigest                               []byte
}

type RegistrationStore interface {
	SavePending(context.Context, PendingRegistration) error
	ConfirmPending(context.Context, string, []byte, string) (domain.User, error)
}

type RegistrationMailer interface {
	SendRegistrationCode(context.Context, string, string) error
}

type Registration struct {
	Store    RegistrationStore
	Mailer   RegistrationMailer
	Password Password
	Tokens   Tokens
	CodeKey  []byte
}

func (s Registration) digest(id, code string) []byte {
	mac := hmac.New(sha256.New, s.CodeKey)
	fmt.Fprintf(mac, "registration:%s:%s", id, code)
	return mac.Sum(nil)
}

func (s Registration) Start(ctx context.Context, name, phone, email, password string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	phone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(phone))
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 254 || len(password) < 12 || len(password) > 128 ||
		name == "" || utf8.RuneCountInString(name) > 180 || strings.IndexFunc(name, unicode.IsControl) >= 0 || !phonePattern.MatchString(phone) {
		return "", domain.ErrValidation
	}
	if len(s.CodeKey) < 32 || s.Mailer == nil {
		return "", ErrDelivery
	}
	hash, err := s.Password.Hash(password)
	if err != nil {
		return "", err
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	p := PendingRegistration{
		ID: id.String(), Email: email, FullName: name, Phone: phone,
		PasswordHash: hash, CodeDigest: s.digest(id.String(), code),
	}
	if err = s.Store.SavePending(ctx, p); err != nil {
		return "", err
	}
	if err = s.Mailer.SendRegistrationCode(ctx, email, code); err != nil {
		return "", ErrDelivery
	}
	return p.ID, nil
}

func (s Registration) Confirm(ctx context.Context, id, code string) (domain.Session, error) {
	if _, err := uuid.Parse(id); err != nil || !codePattern.MatchString(code) {
		return domain.Session{}, domain.ErrValidation
	}
	if len(s.CodeKey) < 32 {
		return domain.Session{}, ErrDelivery
	}
	raw, hash, err := s.Tokens.Opaque()
	if err != nil {
		return domain.Session{}, err
	}
	u, err := s.Store.ConfirmPending(ctx, id, s.digest(id, code), hash)
	if err != nil {
		return domain.Session{}, err
	}
	access, err := s.Tokens.Access(u)
	return domain.Session{Access: access, Refresh: raw, User: u}, err
}
