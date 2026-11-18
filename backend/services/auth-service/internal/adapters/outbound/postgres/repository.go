package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"saya/auth-service/internal/domain"
	"time"
)

type Repository struct{ DB *sqlx.DB }

func (r Repository) Create(ctx context.Context, email, hash string) (domain.User, error) {
	u := domain.User{ID: uuid.NewString(), Email: email, Role: "USER", Password: hash}
	_, e := r.DB.ExecContext(ctx, "INSERT INTO users(id,email,password_hash) VALUES($1,$2,$3)", u.ID, email, hash)
	var pe *pq.Error
	if errors.As(e, &pe) && pe.Code == "23505" {
		return u, domain.ErrConflict
	}
	return u, e
}
func (r Repository) Find(ctx context.Context, email string) (u domain.User, e error) {
	e = r.DB.GetContext(ctx, &u, "SELECT id,email,role,password_hash FROM users WHERE email=$1", email)
	return
}
func (r Repository) Get(ctx context.Context, id string) (u domain.User, e error) {
	e = r.DB.GetContext(ctx, &u, "SELECT id,email,role,password_hash FROM users WHERE id=$1", id)
	return
}
func (r Repository) NewSession(ctx context.Context, id, hash string) error {
	tx, e := r.DB.BeginTxx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = insertSession(ctx, tx, id, hash); e != nil {
		return e
	}
	return tx.Commit()
}
func insertSession(ctx context.Context, tx *sqlx.Tx, id, hash string) error {
	var e error
	sid := uuid.NewString()
	if _, e = tx.ExecContext(ctx, "INSERT INTO sessions(id,user_id,expires_at) VALUES($1,$2,$3)", sid, id, time.Now().Add(30*24*time.Hour)); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO refresh_tokens(hash,session_id) VALUES($1,$2)", hash, sid); e != nil {
		return e
	}
	return nil
}
func (r Repository) Rotate(ctx context.Context, old, next string) (domain.User, error) {
	var u domain.User
	tx, e := r.DB.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		return u, e
	}
	defer tx.Rollback()
	var sid, uid string
	var revoked, consumed bool
	var expires time.Time
	e = tx.QueryRowContext(ctx, "SELECT s.id,s.user_id,s.revoked,s.expires_at,t.consumed FROM sessions s JOIN refresh_tokens t ON t.session_id=s.id WHERE t.hash=$1 FOR UPDATE OF s,t", old).Scan(&sid, &uid, &revoked, &expires, &consumed)
	if errors.Is(e, sql.ErrNoRows) {
		return u, domain.ErrCredentials
	}
	if e != nil {
		return u, e
	}
	if revoked || consumed || !expires.After(time.Now()) {
		if _, e = tx.ExecContext(ctx, "UPDATE sessions SET revoked=true WHERE id=$1", sid); e != nil {
			return u, e
		}
		if e = tx.Commit(); e != nil {
			return u, e
		}
		return u, domain.ErrCredentials
	}
	if _, e = tx.ExecContext(ctx, "UPDATE refresh_tokens SET consumed=true WHERE hash=$1", old); e != nil {
		return u, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO refresh_tokens(hash,session_id) VALUES($1,$2)", next, sid); e != nil {
		return u, e
	}
	if e = tx.GetContext(ctx, &u, "SELECT id,email,role,password_hash FROM users WHERE id=$1", uid); e != nil {
		return u, e
	}
	return u, tx.Commit()
}
func (r Repository) Revoke(ctx context.Context, hash string) error {
	_, e := r.DB.ExecContext(ctx, "UPDATE sessions SET revoked=true WHERE id=(SELECT session_id FROM refresh_tokens WHERE hash=$1)", hash)
	return e
}
