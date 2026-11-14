package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
)

type Argon struct {
	Memory, Iterations uint32
	Threads            uint8
}

func New(memory, iterations uint32, threads uint8) (Argon, error) {
	if memory < 64*1024 || memory > 256*1024 || iterations < 1 || iterations > 6 || threads < 1 || threads > 8 {
		return Argon{}, errors.New("invalid Argon2id configuration")
	}
	return Argon{Memory: memory, Iterations: iterations, Threads: threads}, nil
}
func (p Argon) params() (uint32, uint32, uint8) {
	if p.Memory == 0 {
		return 64 * 1024, 3, 2
	}
	return p.Memory, p.Iterations, p.Threads
}
func (p Argon) Hash(password string) (string, error) {
	m, t, n := p.params()
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return "", e
	}
	hash := argon2.IDKey([]byte(password), salt, t, m, n, 32)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", m, t, n, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func (p Argon) Verify(password, encoded string) bool {
	m, t, n := p.params()
	salt, expected := make([]byte, 16), make([]byte, 32)
	valid := false
	parts := strings.Split(encoded, "$")
	var a, b string
	if len(parts) == 3 && parts[0] == "saya-argon2id-v1" {
		m, t, n = 64*1024, 3, 2
		a, b = parts[1], parts[2]
		valid = true
	}
	if len(parts) == 6 && parts[0] == "" && parts[1] == "argon2id" && parts[2] == "v=19" {
		var pm, pt uint32
		var pn uint8
		if count, e := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &pm, &pt, &pn); e == nil && count == 3 {
			if _, e = New(pm, pt, pn); e == nil {
				m, t, n = pm, pt, pn
				a, b = parts[4], parts[5]
				valid = true
			}
		}
	}
	if valid {
		s, e := base64.RawStdEncoding.DecodeString(a)
		h, f := base64.RawStdEncoding.DecodeString(b)
		if e == nil && f == nil && len(s) == 16 && len(h) == 32 {
			salt, expected = s, h
		} else {
			valid = false
		}
	}
	actual := argon2.IDKey([]byte(password), salt, t, m, n, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1 && valid
}
