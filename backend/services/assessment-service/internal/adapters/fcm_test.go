package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFCMNeutralNotification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Message struct {
				FID                string
				Notification, Data map[string]string
			}
		}
		if e := json.NewDecoder(r.Body).Decode(&payload); e != nil {
			t.Error(e)
		}
		if payload.Message.FID != "installation" || len(payload.Message.Notification) != 2 || len(payload.Message.Data) != 1 || payload.Message.Data["route"] != "weekly_survey" {
			t.Error("unexpected push payload")
		}
	}))
	defer server.Close()
	invalid, err := (&FCM{client: server.Client(), endpoint: server.URL}).Send(context.Background(), "installation", "assessment-id")
	if invalid || err != nil {
		t.Fatal(err)
	}
}
