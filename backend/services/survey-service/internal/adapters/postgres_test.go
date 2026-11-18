package adapters

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"os"
	"saya/survey-service/internal/domain"
	"strings"
	"testing"
)

func TestWeeklySubmissionPersistence(t *testing.T) {
	dsn := os.Getenv("TEST_SURVEY_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_SURVEY_DATABASE_URL required")
	}
	db, e := sqlx.Connect("postgres", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = db.Exec("CREATE SCHEMA " + schema); e != nil {
		t.Fatal(e)
	}
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	if _, e = db.Exec("SET search_path TO " + schema); e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../../migrations/00001_survey.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(strings.Split(string(raw), "-- +goose Down")[0]); e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	r := Repository{DB: db}
	s := domain.Submission{UserID: uuid.NewString(), QuestionnaireID: "saya-week", Version: 2, Consent: true, IdempotencyKey: uuid.NewString(), Ratings: map[string]int32{"housing": 0}, Summary: "Confirmed"}
	id, e := r.Save(ctx, s)
	if e != nil {
		t.Fatal(e)
	}
	other := uuid.NewString()
	if latest, e := r.Latest(ctx, other); e != nil || latest != "" {
		t.Fatal("foreign latest")
	}
	if _, e = r.Input(ctx, id, other); !errors.Is(e, domain.ErrNotFound) {
		t.Fatal("foreign input")
	}
	if latest, e := r.Latest(ctx, s.UserID); e != nil || latest != id {
		t.Fatal("missing latest")
	}
	if repeat, e := r.Save(ctx, s); e != nil || repeat != id {
		t.Fatal("duplicate operation")
	}
	s.Summary = "changed"
	if _, e = r.Save(ctx, s); !errors.Is(e, domain.ErrInvalid) {
		t.Fatal("idempotency conflict missed")
	}
	saved, e := r.Input(ctx, id, s.UserID)
	if e != nil || saved.Summary != "Confirmed" {
		t.Fatal("persisted summary changed")
	}
}
