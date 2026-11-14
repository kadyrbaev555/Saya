package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"log/slog"
	"net/http"
	"os"
	"saya/messaging"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("demo failed", "error", e)
		os.Exit(1)
	}
}
func run() error {
	user := os.Getenv("DEMO_USER_ID")
	if _, e := uuid.Parse(user); e != nil {
		return errors.New("DEMO_USER_ID must be UUID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	db, e := sqlx.ConnectContext(ctx, "postgres", os.Getenv("DATABASE_URL"))
	if e != nil {
		return e
	}
	defer db.Close()
	var exists bool
	if e = db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", user); e != nil || !exists {
		return errors.New("demo user unavailable")
	}
	publisher, e := messaging.Open(os.Getenv("RABBIT_URL"), "saya.demo.v1")
	if e != nil {
		return e
	}
	defer publisher.Close()
	worker, e := messaging.Open(os.Getenv("RABBIT_URL"), "saya.demo.v1")
	if e != nil {
		return e
	}
	defer worker.Close()
	event := messaging.Event{Version: 1, ID: uuid.NewString(), UserID: user, OperationID: uuid.NewString()}
	if e = publisher.Publish(ctx, event); e != nil {
		return e
	}
	return worker.Consume(ctx, func(task context.Context, v messaging.Event) error {
		var processed bool
		if e := db.GetContext(task, &processed, "SELECT EXISTS(SELECT 1 FROM demo_inbox WHERE event_id=$1)", v.ID); e != nil {
			return e
		}
		if processed {
			return nil
		}
		payload, _ := json.Marshal(map[string]any{"channel": "user:" + v.UserID + ":assessments", "data": map[string]any{"event": "demo.completed", "assessment_id": v.OperationID, "status": "completed", "version": 1}})
		req, e := http.NewRequestWithContext(task, http.MethodPost, os.Getenv("CENTRIFUGO_URL")+"/api/publish", bytes.NewReader(payload))
		if e != nil {
			return e
		}
		req.Header.Set("X-API-Key", os.Getenv("CENTRIFUGO_API_KEY"))
		req.Header.Set("Content-Type", "application/json")
		response, e := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if e != nil {
			return e
		}
		defer response.Body.Close()
		body, e := io.ReadAll(io.LimitReader(response.Body, 65536))
		if e != nil {
			return e
		}
		var result struct {
			Error json.RawMessage `json:"error"`
		}
		if response.StatusCode != 200 || json.Unmarshal(body, &result) != nil || len(result.Error) > 0 {
			return errors.New("realtime publish rejected")
		}
		if _, e = db.ExecContext(task, "INSERT INTO demo_inbox(event_id) VALUES($1) ON CONFLICT DO NOTHING", v.ID); e != nil {
			return e
		}
		slog.Info("neutral demo event delivered")
		return nil
	})
}
