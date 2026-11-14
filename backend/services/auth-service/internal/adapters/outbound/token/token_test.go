package token

import (
	"crypto/ed25519"
	"crypto/rand"
	"github.com/golang-jwt/jwt/v5"
	"saya/auth-service/internal/domain"
	"testing"
	"time"
)

func TestTokens(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	p := Provider{Key: key, Kid: "local-1"}
	raw, e := p.Access(domain.User{ID: "user", Role: "USER"})
	if e != nil {
		t.Fatal(e)
	}
	parse := func(s string, k ed25519.PublicKey) error {
		_, e := jwt.Parse(s, func(*jwt.Token) (any, error) { return k, nil }, jwt.WithValidMethods([]string{"EdDSA"}), jwt.WithIssuer("saya-auth"), jwt.WithAudience("saya-api"), jwt.WithExpirationRequired())
		return e
	}
	if e = parse(raw, pub); e != nil {
		t.Fatal(e)
	}
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	if parse(raw, other) == nil {
		t.Fatal("accepted wrong key")
	}
	expired := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": "saya-auth", "aud": "saya-api", "exp": time.Now().Add(-time.Minute).Unix()})
	s, _ := expired.SignedString(key)
	if parse(s, pub) == nil {
		t.Fatal("accepted expired token")
	}
	a, h, e := p.Opaque()
	if e != nil || len(a) != 43 || h == a || h != p.Digest(a) {
		t.Fatal("invalid opaque token")
	}
	b, _, _ := p.Opaque()
	if a == b {
		t.Fatal("repeated random token")
	}
}
