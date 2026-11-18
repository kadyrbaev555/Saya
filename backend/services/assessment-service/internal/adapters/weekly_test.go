package adapters

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"saya/assessment-service/internal/domain"
	"testing"
)

type pushRecorder struct {
	calls int
	fail  bool
}

func (f *pushRecorder) Send(context.Context, string, string) (bool, error) {
	f.calls++
	if f.fail {
		return false, errors.New("temporary")
	}
	return false, nil
}

func checkWeeklyLifecycle(t *testing.T, r Repository, id, user string, signal domain.Signal) {
	t.Helper()
	ctx := context.Background()
	h, e := r.History(ctx, user, "")
	if e != nil || len(h.Entries) != 1 || h.Due || h.NextDue-h.Entries[0].CompletedAt < 604800 || h.NextDue-h.Entries[0].CompletedAt > 604801 {
		t.Fatal("weekly history/schedule", e)
	}
	if foreign, e := r.History(ctx, uuid.NewString(), ""); e != nil || len(foreign.Entries) != 0 || foreign.NextDue != 0 {
		t.Fatal("ownership", e)
	}
	const device = "test-device"
	if e = r.SetDevice(ctx, user, device, true); e != nil {
		t.Fatal(e)
	}
	sender := &pushRecorder{}
	if e = r.SendDue(ctx, sender); e != nil || sender.calls != 0 {
		t.Fatal("early push", e)
	}
	// Advance the disposable fixture to the seventh day, leaving production clocks untouched.
	for _, q := range []string{
		"DELETE FROM notifications WHERE id=$1",
		"UPDATE push_devices SET enabled_at=now()-interval '8 days' WHERE user_id=(SELECT user_id FROM assessments WHERE id=$1)",
		"UPDATE weekly_reminders SET due_at=now()-interval '1 second' WHERE assessment_id=$1",
	} {
		if _, e = r.DB.Exec(q, id); e != nil {
			t.Fatal(e)
		}
	}
	h, e = r.History(ctx, user, "")
	if e != nil || !h.Due {
		t.Fatal("missing due card", e)
	}
	for i := 0; i < 2; i++ {
		if e = r.AnnounceDue(ctx); e != nil {
			t.Fatal(e)
		}
	}
	var count int
	if e = r.DB.Get(&count, "SELECT count(*) FROM notifications WHERE status='reminder' AND id=$1", id); e != nil || count != 1 {
		t.Fatal("duplicate reminder event", count, e)
	}
	sender.fail = true
	if e = r.SendDue(ctx, sender); e == nil {
		t.Fatal("expected retry")
	}
	if e = r.SendDue(ctx, sender); e != nil || sender.calls != 1 {
		t.Fatal("backoff missing", e)
	}
	if _, e = r.DB.Exec("UPDATE push_devices SET retry_at=now() WHERE token=$1", device); e != nil {
		t.Fatal(e)
	}
	sender.fail = false
	for i := 0; i < 2; i++ {
		if e = r.SendDue(ctx, sender); e != nil {
			t.Fatal(e)
		}
	}
	if sender.calls != 2 {
		t.Fatal("duplicate push", sender.calls)
	}
	next := uuid.NewString()
	signal.Ratings["housing"] = 2
	if e = r.Create(ctx, next, user, signal); e != nil {
		t.Fatal(e)
	}
	if e = r.Complete(ctx, next, user, nil, "self-report", false); e != nil {
		t.Fatal(e)
	}
	h, e = r.History(ctx, user, "")
	if e != nil || h.Due || len(h.Entries) != 2 || h.Entries[0].ID != next || *h.Entries[0].Result.Percentage != 60 {
		t.Fatal("repeat result/schedule", e)
	}
	if older, e := r.History(ctx, user, next); e != nil || len(older.Entries) != 1 || older.Entries[0].ID != id {
		t.Fatal("history cursor", e)
	}
	if e = r.SendDue(ctx, sender); e != nil || sender.calls != 2 {
		t.Fatal("old reminder still eligible", e)
	}
	if e = r.SetDevice(ctx, user, device, false); e != nil {
		t.Fatal(e)
	}
	if e = r.DB.Get(&count, "SELECT count(*) FROM push_devices"); e != nil || count != 0 {
		t.Fatal("device opt-out", e)
	}
}
