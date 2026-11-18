package postgres

import (
	"context"
	"crypto/hmac"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"saya/auth-service/internal/application"
	"saya/auth-service/internal/domain"
)

func (r Repository) SavePending(ctx context.Context, p application.PendingRegistration) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DELETE FROM pending_registrations WHERE expires_at <= now()"); err != nil {
		return err
	}
	var exists bool
	if err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", p.Email); err != nil {
		return err
	}
	if exists {
		return domain.ErrConflict
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO pending_registrations(id,email,full_name,phone,password_hash,code_digest)
 VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(email) DO UPDATE SET
 id=excluded.id,full_name=excluded.full_name,phone=excluded.phone,password_hash=excluded.password_hash,
 code_digest=excluded.code_digest,sent_at=excluded.sent_at,expires_at=excluded.expires_at,attempts_left=5
 WHERE pending_registrations.sent_at <= now()-interval '1 minute'`, p.ID, p.Email, p.FullName, p.Phone, p.PasswordHash, p.CodeDigest)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return application.ErrResendTooSoon
	}
	return tx.Commit()
}

func (r Repository) ConfirmPending(ctx context.Context, id string, digest []byte, refreshHash string) (domain.User, error) {
	var u domain.User
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	var expected []byte
	err = tx.QueryRowxContext(ctx, `SELECT code_digest FROM pending_registrations
 WHERE id=$1 AND expires_at>clock_timestamp() AND attempts_left>0 FOR UPDATE`, id).Scan(&expected)
	if errors.Is(err, sql.ErrNoRows) {
		return u, domain.ErrCredentials
	}
	if err != nil {
		return u, err
	}
	if !hmac.Equal(expected, digest) {
		if _, err = tx.ExecContext(ctx, "UPDATE pending_registrations SET attempts_left=attempts_left-1 WHERE id=$1", id); err != nil {
			return u, err
		}
		if err = tx.Commit(); err != nil {
			return u, err
		}
		return u, domain.ErrCredentials
	}
	err = tx.QueryRowxContext(ctx, `INSERT INTO users(id,email,password_hash,full_name,phone)
 SELECT $2,email,password_hash,full_name,phone FROM pending_registrations WHERE id=$1
 ON CONFLICT(email) DO NOTHING RETURNING id,email,role`, id, uuid.NewString()).Scan(&u.ID, &u.Email, &u.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return u, domain.ErrConflict
	}
	if err != nil {
		return u, err
	}
	if err = insertSession(ctx, tx, u.ID, refreshHash); err != nil {
		return u, err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM pending_registrations WHERE id=$1", id); err != nil {
		return u, err
	}
	return u, tx.Commit()
}
