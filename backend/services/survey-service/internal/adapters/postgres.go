package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"saya/messaging"
	"saya/survey-service/internal/domain"
)

type Repository struct{ DB *sqlx.DB }

func (r Repository) Latest(ctx context.Context, user string) (string, error) {
	var id string
	err := r.DB.GetContext(ctx, &id, "SELECT id FROM submissions WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1", user)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}
func (r Repository) Save(ctx context.Context, v domain.Submission) (string, error) {
	tx, e := r.DB.BeginTxx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	id := uuid.NewString()
	body, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	var actual string
	e = tx.QueryRowContext(ctx, "INSERT INTO submissions(id,user_id,idempotency_key,payload) VALUES($1,$2,$3,$4) ON CONFLICT(user_id,idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key RETURNING id", id, v.UserID, v.IdempotencyKey, string(body)).Scan(&actual)
	if e != nil {
		return "", e
	}
	if actual != id {
		var old string
		if e = tx.GetContext(ctx, &old, "SELECT payload::text FROM submissions WHERE id=$1", actual); e != nil {
			return "", e
		}
		var existing domain.Submission
		if e = json.Unmarshal([]byte(old), &existing); e != nil {
			return "", e
		}
		a, _ := json.Marshal(existing)
		if string(a) != string(body) {
			return "", domain.ErrInvalid
		}
	} else {
		event, _ := json.Marshal(messaging.Event{Version: 1, ID: uuid.NewString(), UserID: v.UserID, OperationID: id})
		if _, e = tx.ExecContext(ctx, "INSERT INTO outbox(id,payload) VALUES($1,$2)", uuid.NewString(), string(event)); e != nil {
			return "", e
		}
	}
	return actual, tx.Commit()
}
func (r Repository) Input(ctx context.Context, id, user string) (domain.Submission, error) {
	var v domain.Submission
	var body []byte
	if e := r.DB.GetContext(ctx, &body, "SELECT payload FROM submissions WHERE id=$1 AND user_id=$2", id, user); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return v, domain.ErrNotFound
		}
		return v, e
	}
	e := json.Unmarshal(body, &v)
	return v, e
}
