package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"saya/ai-service/internal/domain"
	"testing"
	"time"
)

func TestHTTPProvider(t *testing.T) {
	for _, test := range []struct {
		name, body string
		valid      bool
	}{{"valid", `{"recommendations":["choose_safe_device"]}`, true}, {"unknown", `{"recommendations":["diagnosis"]}`, false}, {"extra", `{"recommendations":["choose_safe_device"],"score":42}`, false}, {"trailing", `{"recommendations":["choose_safe_device"]}{}`, false}, {"malformed", `not-json`, false}} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer demo" {
					t.Error("missing auth")
				}
				w.Write([]byte(test.body))
			}))
			defer server.Close()
			p := HTTPProvider{Endpoint: server.URL, APIKey: "demo", Client: server.Client()}
			_, e := p.Generate(context.Background(), domain.Input{ExternalConsent: true})
			if (e == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, e)
			}
			if _, e = p.Generate(context.Background(), domain.Input{}); e == nil {
				t.Fatal("missing consent accepted")
			}
		})
	}
}
func TestHTTPTimeout(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	}))
	defer server.Close()
	p := HTTPProvider{Endpoint: server.URL, Client: server.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, e := p.Generate(ctx, domain.Input{ExternalConsent: true}); e == nil {
		t.Fatal("timeout ignored")
	}
}
