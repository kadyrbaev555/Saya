package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"saya/assessment-service/internal/domain"
	"saya/messaging"
)

type Repository struct{ DB *sqlx.DB }

func (r Repository) Create(ctx context.Context, id, user string, signal domain.Signal) error {
	tx, e := r.DB.BeginTxx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	input, _ := json.Marshal(signal)
	result, _ := json.Marshal(domain.Build(id, "pending", signal, nil, ""))
	res, e := tx.ExecContext(ctx, "INSERT INTO assessments(id,user_id,input,result,status) VALUES($1,$2,$3,$4,'pending') ON CONFLICT DO NOTHING", id, user, string(input), string(result))
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 1 {
		event, _ := json.Marshal(messaging.Event{Version: 1, ID: uuid.NewString(), UserID: user, OperationID: id})
		if _, e = tx.ExecContext(ctx, "INSERT INTO outbox(id,payload) VALUES($1,$2)", uuid.NewString(), string(event)); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (r Repository) Input(ctx context.Context, id, user string) (domain.Signal, error) {
	var v domain.Signal
	var body []byte
	if e := r.DB.GetContext(ctx, &body, "SELECT input FROM assessments WHERE id=$1 AND user_id=$2", id, user); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return v, domain.ErrNotFound
		}
		return v, e
	}
	e := json.Unmarshal(body, &v)
	return v, e
}
func (r Repository) Get(ctx context.Context, id, user string) (domain.Result, error) {
	var v domain.Result
	var body []byte
	if e := r.DB.GetContext(ctx, &body, "SELECT result FROM assessments WHERE id=$1 AND user_id=$2", id, user); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return v, domain.ErrNotFound
		}
		return v, e
	}
	e := json.Unmarshal(body, &v)
	return v, e
}
func (r Repository) Complete(ctx context.Context, id, user string, codes []string, provider string, failed bool) error {
	tx, e := r.DB.BeginTxx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var input []byte
	var current string
	if e = tx.QueryRowContext(ctx, "SELECT input,status FROM assessments WHERE id=$1 AND user_id=$2 FOR UPDATE", id, user).Scan(&input, &current); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return e
	}
	if current != "pending" {
		return tx.Commit()
	}
	var signal domain.Signal
	if e = json.Unmarshal(input, &signal); e != nil {
		return e
	}
	state := "completed"
	if failed {
		state = "failed"
		codes = nil
		provider = "unavailable"
	}
	result, _ := json.Marshal(domain.Build(id, state, signal, codes, provider))
	if _, e = tx.ExecContext(ctx, "UPDATE assessments SET result=$1,status=$2,completed_at=CASE WHEN $2='completed' THEN clock_timestamp() END WHERE id=$3", string(result), state, id); e != nil {
		return e
	}
	if !failed && signal.SourceVersion == 2 {
		_, e = tx.ExecContext(ctx, `INSERT INTO weekly_reminders(user_id,assessment_id,due_at)
    SELECT user_id,id,completed_at+interval '168 hours' FROM assessments WHERE id=$1
    ON CONFLICT(user_id) DO UPDATE SET assessment_id=excluded.assessment_id,due_at=excluded.due_at,announced=false
    WHERE (excluded.due_at,excluded.assessment_id)>(weekly_reminders.due_at,weekly_reminders.assessment_id)`, id)
		if e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO notifications(id,user_id,status) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", id, user, state); e != nil {
		return e
	}
	return tx.Commit()
}
