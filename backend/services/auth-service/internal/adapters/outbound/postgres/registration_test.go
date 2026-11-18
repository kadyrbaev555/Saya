package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"saya/auth-service/internal/adapters/outbound/password"
	"saya/auth-service/internal/adapters/outbound/token"
	"saya/auth-service/internal/application"
	"saya/auth-service/internal/domain"
)

type inbox struct {
	code string
	fail bool
}

func (i *inbox) SendRegistrationCode(_ context.Context, _ string, code string) error {
	if i.fail {
		return errors.New("delivery failed")
	}
	i.code = code
	return nil
}

func TestRegistrationIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL requires isolated migrated database")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := Repository{DB: db}
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tokens := token.Provider{Key: key, Kid: "test"}
	passwords, err := password.New(65536, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	mail := &inbox{}
	reg := application.Registration{Store: repo, Mailer: mail, Password: passwords, Tokens: tokens, CodeKey: key.Seed()}
	ctx := context.Background()
	address := uuid.NewString() + "@example.test"
	defer func() {
		db.Exec("DELETE FROM pending_registrations WHERE email=$1", address)
		db.Exec("DELETE FROM refresh_tokens WHERE session_id IN(SELECT s.id FROM sessions s JOIN users u ON u.id=s.user_id WHERE u.email=$1)", address)
		db.Exec("DELETE FROM sessions WHERE user_id IN(SELECT id FROM users WHERE email=$1)", address)
		db.Exec("DELETE FROM users WHERE email=$1", address)
	}()
	start := func() (string, error) {
		return reg.Start(ctx, "Test Person", "+7 (777) 123-45-67", address, "test-password-with-entropy")
	}
	mail.fail = true
	if _, err = start(); !errors.Is(err, application.ErrDelivery) {
		t.Fatal("delivery error", err)
	}
	if _, err = start(); !errors.Is(err, application.ErrResendTooSoon) {
		t.Fatal("cooldown", err)
	}
	mail.fail = false
	age := func() {
		t.Helper()
		if _, err = db.Exec("UPDATE pending_registrations SET sent_at=now()-interval '2 minutes' WHERE email=$1", address); err != nil {
			t.Fatal(err)
		}
	}
	age()
	id, err := start()
	if err != nil {
		t.Fatal(err)
	}
	login := application.Service{Repo: repo, Password: passwords, Tokens: tokens}
	if _, err = login.Login(ctx, address, "test-password-with-entropy"); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("login before confirmation", err)
	}
	wrong := "000000"
	if wrong == mail.code {
		wrong = "000001"
	}
	for i := 0; i < 5; i++ {
		if _, err = reg.Confirm(ctx, id, wrong); !errors.Is(err, domain.ErrCredentials) {
			t.Fatal("incorrect code", err)
		}
	}
	if _, err = reg.Confirm(ctx, id, mail.code); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("attempt limit", err)
	}
	age()
	id, err = start()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE pending_registrations SET expires_at=now() WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	if _, err = reg.Confirm(ctx, id, mail.code); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("expired code", err)
	}
	id, err = start()
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, e := reg.Confirm(ctx, id, mail.code); results <- e }()
	}
	a, b := <-results, <-results
	if !((a == nil && errors.Is(b, domain.ErrCredentials)) || (b == nil && errors.Is(a, domain.ErrCredentials))) {
		t.Fatalf("concurrent confirmations: %v %v", a, b)
	}
	var count int
	if err = db.Get(&count, "SELECT count(*) FROM sessions s JOIN users u ON u.id=s.user_id WHERE u.email=$1 AND u.phone=$2 AND u.full_name=$3", address, "+77771234567", "Test Person"); err != nil || count != 1 {
		t.Fatal("expected one account session", count, err)
	}
	if _, err = login.Login(ctx, address, "test-password-with-entropy"); err != nil {
		t.Fatal(err)
	}
}

func TestLoginAccountWithoutRegistrationProfile(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL requires isolated migrated database")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := Repository{DB: db}
	address := uuid.NewString() + "@example.test"
	passwords, err := password.New(65536, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := passwords.Hash("existing-demo-password")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	u, err := repo.Create(ctx, address, hash)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Exec("DELETE FROM refresh_tokens WHERE session_id IN (SELECT id FROM sessions WHERE user_id=$1)", u.ID)
		db.Exec("DELETE FROM sessions WHERE user_id=$1", u.ID)
		db.Exec("DELETE FROM users WHERE id=$1", u.ID)
	}()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	service := application.Service{Repo: repo, Password: passwords, Tokens: token.Provider{Key: key, Kid: "test"}}
	result, err := service.Login(ctx, address, "existing-demo-password")
	if err != nil || result.User.ID != u.ID {
		t.Fatal("existing user login failed", err)
	}
}
