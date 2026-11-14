package platform

import (
	"context"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"saya/messaging"
)

// Deliver holds a row lock until confirmed publication. Replays remain possible after a crash.
func Deliver(ctx context.Context, db *sqlx.DB, send func(context.Context, messaging.Event) error) error {
	tx, e := db.BeginTxx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, "SELECT id,payload FROM outbox ORDER BY created_at LIMIT 20 FOR UPDATE SKIP LOCKED")
	if e != nil {
		return e
	}
	type item struct {
		id   string
		body []byte
	}
	var items []item
	for rows.Next() {
		var v item
		if e = rows.Scan(&v.id, &v.body); e != nil {
			rows.Close()
			return e
		}
		items = append(items, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, v := range items {
		var event messaging.Event
		if e = json.Unmarshal(v.body, &event); e != nil {
			return e
		}
		if e = send(ctx, event); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "DELETE FROM outbox WHERE id=$1", v.id); e != nil {
			return e
		}
	}
	return tx.Commit()
}
