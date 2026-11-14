package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"os"
	"saya/auth-service/internal/domain"
	"sync"
	"testing"
)

func TestRotationIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required; use isolated migrated database")
	}
	db, e := sqlx.Connect("postgres", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	r := Repository{DB: db}
	ctx := context.Background()
	u, e := r.Create(ctx, uuid.NewString()+"@example.test", "hash")
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		db.Exec("DELETE FROM refresh_tokens WHERE session_id IN(SELECT id FROM sessions WHERE user_id=$1)", u.ID)
		db.Exec("DELETE FROM sessions WHERE user_id=$1", u.ID)
		db.Exec("DELETE FROM users WHERE id=$1", u.ID)
	}()
	old := uuid.NewString()
	if e = r.NewSession(ctx, u.ID, old); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := r.Rotate(ctx, old, uuid.NewString()); out <- e }()
	}
	wg.Wait()
	close(out)
	success, denied := 0, 0
	for e := range out {
		if e == nil {
			success++
		} else if errors.Is(e, domain.ErrCredentials) {
			denied++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || denied != 1 {
		t.Fatalf("success=%d denied=%d", success, denied)
	}
	var revoked bool
	if e = db.Get(&revoked, "SELECT revoked FROM sessions WHERE user_id=$1", u.ID); e != nil || !revoked {
		t.Fatal("replay must revoke family", e)
	}
}
