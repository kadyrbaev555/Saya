package adapters

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"os"
	"saya/assessment-service/internal/domain"
	"strings"
	"testing"
)

func TestAssessmentPersistence(t *testing.T) {
	dsn := os.Getenv("TEST_ASSESSMENT_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_ASSESSMENT_DATABASE_URL required")
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
	migration, e := os.ReadFile("../../migrations/00001_assessment.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(strings.Split(string(migration), "-- +goose Down")[0]); e != nil {
		t.Fatal(e)
	}
	r := Repository{DB: db}
	ctx := context.Background()
	id, user := uuid.NewString(), uuid.NewString()
	signal := domain.Signal{Type: "survey", SourceVersion: 1, ImmediateThreat: true}
	for i := 0; i < 2; i++ {
		if e = r.Create(ctx, id, user, signal); e != nil {
			t.Fatal(e)
		}
	}
	var jobs int
	if e = db.Get(&jobs, "SELECT count(*) FROM outbox"); e != nil || jobs != 1 {
		t.Fatal("duplicate job", jobs, e)
	}
	pending, e := r.Get(ctx, id, user)
	if e != nil || pending.Status != "pending" || !pending.ImmediateThreat {
		t.Fatal("pending rule missing", e)
	}
	if _, e = r.Get(ctx, id, uuid.NewString()); e != domain.ErrNotFound {
		t.Fatal("ownership failed")
	}
	if e = r.Complete(ctx, id, user, nil, "unavailable", true); e != nil {
		t.Fatal(e)
	}
	if e = r.Complete(ctx, id, user, []string{"choose_safe_device"}, "demonstration-mock", false); e != nil {
		t.Fatal(e)
	}
	failed, e := r.Get(ctx, id, user)
	if e != nil || failed.Status != "failed" || len(failed.Recommendations) != 1 {
		t.Fatal("failed status/rule/idempotency failed", e)
	}
	if e = db.Get(&jobs, "SELECT count(*) FROM notifications"); e != nil || jobs != 1 {
		t.Fatal("duplicate notification", e)
	}
}
