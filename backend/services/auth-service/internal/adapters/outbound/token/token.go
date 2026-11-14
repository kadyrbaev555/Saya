package token

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"github.com/golang-jwt/jwt/v5"
	"saya/auth-service/internal/domain"
	"time"
)

type Provider struct {
	Key ed25519.PrivateKey
	Kid string
}

func (p Provider) Access(u domain.User) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"sub": u.ID, "role": u.Role, "iss": "saya-auth", "aud": "saya-api", "iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix()})
	t.Header["kid"] = p.Kid
	return t.SignedString(p.Key)
}
func (p Provider) Opaque() (string, string, error) {
	b := make([]byte, 32)
	_, e := rand.Read(b)
	s := base64.RawURLEncoding.EncodeToString(b)
	return s, p.Digest(s), e
}
func (Provider) Digest(s string) string {
	h := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
