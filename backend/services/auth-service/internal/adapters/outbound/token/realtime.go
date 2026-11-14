package token

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Realtime struct{ Secret []byte }

func (p Realtime) Sign(user, channel string) (string, int64, error) {
	exp := time.Now().Add(5 * time.Minute).Unix()
	claims := jwt.MapClaims{"sub": user, "exp": exp}
	if channel != "" {
		claims["channel"] = channel
	}
	s, e := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(p.Secret)
	return s, exp, e
}
