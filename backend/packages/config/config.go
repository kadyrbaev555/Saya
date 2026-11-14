// Package config reads dotenv files without modifying process-wide environment.
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Reader struct {
	values map[string]string
	err    error
}

func Open() (*Reader, error) {
	path, explicit := os.LookupEnv("ENV_FILE")
	if !explicit {
		path = ".env"
	}
	values, err := godotenv.Read(path)
	if err != nil && (explicit || !errors.Is(err, os.ErrNotExist)) {
		return nil, errors.New("cannot read or parse ENV_FILE")
	}
	return &Reader{values: values}, nil
}
func (r *Reader) String(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	if v, ok := r.values[key]; ok {
		return v
	}
	return fallback
}
func (r *Reader) Invalid(key string) {
	if r.err == nil {
		r.err = fmt.Errorf("invalid configuration: %s", key)
	}
}
func (r *Reader) Required(key string) string {
	v := r.String(key, "")
	if strings.TrimSpace(v) == "" {
		r.Invalid(key)
	}
	return v
}
func (r *Reader) Bool(key string, fallback string) bool {
	v, e := strconv.ParseBool(r.String(key, fallback))
	if e != nil {
		r.Invalid(key)
	}
	return v
}
func (r *Reader) Uint(key, fallback string, min, max uint64) uint64 {
	v, e := strconv.ParseUint(r.String(key, fallback), 10, 64)
	if e != nil || v < min || v > max {
		r.Invalid(key)
	}
	return v
}
func (r *Reader) URL(key string, schemes ...string) string {
	v := r.Required(key)
	u, e := url.Parse(v)
	valid := false
	if e == nil && u.Hostname() != "" && u.Fragment == "" {
		for _, s := range schemes {
			if u.Scheme == s {
				valid = true
			}
		}
	}
	if !valid {
		r.Invalid(key)
	}
	return v
}
func (r *Reader) Address(key, fallback string) string {
	v := r.String(key, fallback)
	host, port, e := net.SplitHostPort(v)
	n, pe := strconv.Atoi(port)
	if e != nil || host == "" || pe != nil || n < 1 || n > 65535 {
		r.Invalid(key)
	}
	return v
}
func (r *Reader) Key(key string) string {
	v := r.Required(key)
	b, e := base64.RawURLEncoding.DecodeString(v)
	if e != nil || len(b) != 32 {
		r.Invalid(key)
	}
	return v
}
func (r *Reader) Err() error { return r.err }
