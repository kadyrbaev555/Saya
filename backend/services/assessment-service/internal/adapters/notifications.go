package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

func (r Repository) Notify(ctx context.Context, url, key string) error {
	tx, e := r.DB.BeginTxx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, "SELECT id,user_id,status FROM notifications ORDER BY created_at LIMIT 20 FOR UPDATE SKIP LOCKED")
	if e != nil {
		return e
	}
	type notification struct{ id, user, status string }
	var items []notification
	for rows.Next() {
		var v notification
		if e = rows.Scan(&v.id, &v.user, &v.status); e != nil {
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
		body, _ := json.Marshal(map[string]any{"channel": "user:" + v.user + ":assessments", "data": map[string]any{"event": "assessment." + v.status, "assessment_id": v.id, "status": v.status, "version": 1}})
		req, e := http.NewRequestWithContext(ctx, "POST", url+"/api/publish", bytes.NewReader(body))
		if e != nil {
			return e
		}
		req.Header.Set("X-API-Key", key)
		req.Header.Set("Content-Type", "application/json")
		resp, e := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if e != nil {
			return e
		}
		data, e := io.ReadAll(io.LimitReader(resp.Body, 65536))
		resp.Body.Close()
		var result struct {
			Error json.RawMessage `json:"error"`
		}
		if e != nil || resp.StatusCode != 200 || json.Unmarshal(data, &result) != nil || len(result.Error) > 0 {
			return errors.New("notification pending")
		}
		if _, e = tx.ExecContext(ctx, "DELETE FROM notifications WHERE id=$1", v.id); e != nil {
			return e
		}
	}
	return tx.Commit()
}
