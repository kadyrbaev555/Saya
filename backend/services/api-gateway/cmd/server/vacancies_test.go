package main

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type vacancyTransport func(*http.Request) (*http.Response, error)

func (f vacancyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestVacancySearch(t *testing.T) {
	for _, tc := range []struct {
		name, body, upstream string
		upstreamCode, want   int
	}{
		{"found", `{"text":" Go ","area":"160"}`, `{"items":[{"name":"Go engineer","employer":{"name":"Company"},"alternate_url":"https://almaty.hh.kz/vacancy/123"}]}`, 200, 200},
		{"empty", `{"text":"Go","area":"160"}`, `{"items":[]}`, 200, 200},
		{"denied", `{"text":"Go","area":"160"}`, "", 403, 503},
		{"invalid results", `{"text":"Go","area":"160"}`, `{"items":[{"name":"Go","alternate_url":"https://example.org/vacancy/123"}]}`, 200, 502},
		{"empty query", `{"text":" ","area":"160"}`, "", 0, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: vacancyTransport(func(r *http.Request) (*http.Response, error) {
				if tc.upstreamCode == 0 {
					t.Fatal("unexpected upstream request")
				}
				if r.URL.Host != "api.hh.ru" || r.URL.Query().Get("area") != "160" || r.URL.Query().Get("text") != "Go" || r.Header.Get("Authorization") != "Bearer test" {
					t.Fatal("unexpected upstream parameters")
				}
				return &http.Response{StatusCode: tc.upstreamCode, Body: io.NopCloser(strings.NewReader(tc.upstream))}, nil
			})}
			router := gin.New()
			router.POST("/search", vacancySearch(client, "test"))
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/search", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status %d, want %d", w.Code, tc.want)
			}
			if tc.name == "found" && !strings.Contains(w.Body.String(), "Go engineer") {
				t.Fatal("missing vacancy")
			}
			if tc.name == "empty" && w.Body.String() != `{"items":[]}` {
				t.Fatal("missing empty list")
			}
		})
	}
}
