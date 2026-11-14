// bootstrap-admin accepts a JSON {email,password} on stdin, never command arguments.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"net/mail"
	"os"
	"saya/auth-service/internal/adapters/outbound/password"
	"strings"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "admin bootstrap failed:", e)
		os.Exit(1)
	}
}
func run() error {
	var input struct{ Email, Password string }
	if e := json.NewDecoder(io.LimitReader(os.Stdin, 4096)).Decode(&input); e != nil {
		return errors.New("expected bounded JSON on stdin")
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	a, e := mail.ParseAddress(input.Email)
	if e != nil || a.Address != input.Email || len(input.Password) < 12 || len(input.Password) > 128 {
		return errors.New("invalid credentials")
	}
	h, e := (password.Argon{}).Hash(input.Password)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, e := sqlx.ConnectContext(ctx, "postgres", os.Getenv("DATABASE_URL"))
	if e != nil {
		return errors.New("database unavailable")
	}
	defer db.Close()
	tx, e := db.BeginTxx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(734219)"); e != nil {
		return e
	}
	var exists bool
	if e = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM users WHERE role='ADMIN')"); e != nil {
		return e
	}
	if exists {
		return errors.New("initial administrator already exists")
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO users(id,email,password_hash,role) VALUES($1,$2,$3,'ADMIN')", uuid.NewString(), input.Email, h); e != nil {
		return errors.New("account unavailable")
	}
	return tx.Commit()
}
