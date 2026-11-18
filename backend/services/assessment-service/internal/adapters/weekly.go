package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"saya/assessment-service/internal/domain"
)

// History is scoped to the authenticated owner, including the page cursor.
func (r Repository) History(ctx context.Context, user, before string) (domain.WeeklyHistory, error) {
	var h domain.WeeklyHistory
	rows, err := r.DB.QueryContext(ctx, `SELECT id,floor(extract(epoch from completed_at))::bigint,result
 FROM assessments a WHERE user_id=$1 AND completed_at IS NOT NULL AND
 (NULLIF($2,'') IS NULL OR (completed_at,id)<(SELECT completed_at,id FROM assessments WHERE id=NULLIF($2,'')::uuid AND user_id=$1))
 ORDER BY completed_at DESC,id DESC LIMIT 21`, user, before)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		if len(h.Entries) == 20 {
			h.NextCursor = h.Entries[19].ID
			break
		}
		var v domain.WeeklyEntry
		var data []byte
		if err = rows.Scan(&v.ID, &v.CompletedAt, &data); err != nil {
			rows.Close()
			return h, err
		}
		if err = json.Unmarshal(data, &v.Result); err != nil {
			rows.Close()
			return h, err
		}
		h.Entries = append(h.Entries, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return h, err
	}
	err = r.DB.QueryRowContext(ctx, "SELECT ceil(extract(epoch from due_at))::bigint,due_at<=now() FROM weekly_reminders WHERE user_id=$1", user).Scan(&h.NextDue, &h.Due)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return h, err
}

func (r Repository) SetDevice(ctx context.Context, user, token string, enabled bool) error {
	if !enabled {
		_, err := r.DB.ExecContext(ctx, "DELETE FROM push_devices WHERE user_id=$1 AND token=$2", user, token)
		return err
	}
	_, err := r.DB.ExecContext(ctx, `INSERT INTO push_devices(token,user_id) VALUES($1,$2)
 ON CONFLICT(token) DO UPDATE SET user_id=excluded.user_id,enabled_at=now(),retry_at=now(),last_assessment=NULL
 WHERE push_devices.user_id<>excluded.user_id`, token, user)
	return err
}

func (r Repository) AnnounceDue(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, `WITH due AS (
 SELECT user_id,assessment_id FROM weekly_reminders WHERE due_at<=now() AND NOT announced
 ORDER BY due_at LIMIT 50 FOR UPDATE SKIP LOCKED
 ),queued AS (
 INSERT INTO notifications(id,user_id,status) SELECT assessment_id,user_id,'reminder' FROM due
 ON CONFLICT DO NOTHING RETURNING id
 ) UPDATE weekly_reminders SET announced=true FROM queued WHERE assessment_id=queued.id`)
	return err
}

type PushSender interface {
	Send(context.Context, string, string) (invalid bool, err error)
}

func (r Repository) SendDue(ctx context.Context, s PushSender) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var token, id string
	err = tx.QueryRowContext(ctx, `SELECT d.token,w.assessment_id FROM weekly_reminders w JOIN push_devices d USING(user_id)
 WHERE w.due_at<=now() AND d.enabled_at<=w.due_at AND d.retry_at<=now()
 AND d.last_assessment IS DISTINCT FROM w.assessment_id
 ORDER BY w.due_at LIMIT 1 FOR UPDATE OF w,d SKIP LOCKED`).Scan(&token, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	invalid, sendErr := s.Send(ctx, token, id)
	switch {
	case invalid:
		_, err = tx.ExecContext(ctx, "DELETE FROM push_devices WHERE token=$1", token)
	case sendErr != nil:
		_, err = tx.ExecContext(ctx, "UPDATE push_devices SET retry_at=now()+interval '10 minutes' WHERE token=$1", token)
	default:
		_, err = tx.ExecContext(ctx, "UPDATE push_devices SET last_assessment=$1 WHERE token=$2", id, token)
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return sendErr
}
