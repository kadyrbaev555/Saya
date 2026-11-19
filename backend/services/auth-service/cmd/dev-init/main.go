package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func run() error {
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		return e
	}
	values := map[string]string{"JWT_SEED": base64.RawURLEncoding.EncodeToString(key.Seed()), "JWT_PUBLIC": base64.RawURLEncoding.EncodeToString(pub)}
	for _, name := range []string{"DB_PASSWORD", "DB_ADMIN_PASSWORD", "SURVEY_DB_PASSWORD", "ASSESSMENT_DB_PASSWORD", "RABBIT_PASSWORD", "CENTRIFUGO_TOKEN_SECRET", "CENTRIFUGO_API_KEY"} {
		b := make([]byte, 32)
		if _, e = rand.Read(b); e != nil {
			return e
		}
		values[name] = base64.RawURLEncoding.EncodeToString(b)
	}
	f, e := os.OpenFile("deployments/local/.env", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	for k, v := range values {
		if _, e = fmt.Fprintf(f, "%s=%s\n", k, v); e != nil {
			f.Close()
			return e
		}
	}
	return f.Close()
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "dev-init failed:", e)
		os.Exit(1)
	}
}
