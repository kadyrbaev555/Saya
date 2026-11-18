package adapters

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"saya/ai-service/internal/domain"
	"strings"
	"testing"
)

type geminiTransport func(*http.Request) (*http.Response, error)

func (f geminiTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestGemini(t *testing.T) {
	for _, test := range []struct {
		name, text, finish, block string
		status                    int
		valid                     bool
	}{
		{"valid", `{"recommendations":["choose_safe_device"]}`, "STOP", "", 200, true},
		{"unknown enum", `{"recommendations":["diagnosis"]}`, "STOP", "", 200, false},
		{"extra field", `{"recommendations":["choose_safe_device"],"score":1}`, "STOP", "", 200, false},
		{"trailing", `{"recommendations":["choose_safe_device"]}{}`, "STOP", "", 200, false},
		{"incomplete", `{"recommendations":["choose_safe_device"]}`, "MAX_TOKENS", "", 200, false},
		{"blocked", "", "", "SAFETY", 200, false},
		{"rate limited", "", "", "", 429, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			p := Gemini{APIKey: "test-secret", Model: "test-model"}
			p.Client = &http.Client{Transport: geminiTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "generativelanguage.googleapis.com" || r.URL.RawQuery != "" || r.Header.Get("x-goog-api-key") != "test-secret" {
					t.Fatal("invalid authentication or target")
				}
				requestBody, _ := io.ReadAll(r.Body)
				if strings.Contains(string(requestBody), "test-secret") || strings.Contains(string(requestBody), "external_consent") {
					t.Fatal("secret/consent flag leaked into model content")
				}
				var request map[string]json.RawMessage
				if json.Unmarshal(requestBody, &request) != nil || request["generationConfig"] == nil {
					t.Fatal("structured config missing")
				}
				body, _ := json.Marshal(map[string]any{"promptFeedback": map[string]string{"blockReason": test.block}, "candidates": []any{map[string]any{"finishReason": test.finish, "content": map[string]any{"parts": []any{map[string]string{"text": test.text}}}}}})
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
			})}
			_, err := p.Generate(context.Background(), domain.Input{ExternalConsent: true})
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
			_, err = p.Generate(context.Background(), domain.Input{})
			if err == nil || calls != 1 {
				t.Fatal("missing consent called provider")
			}
		})
	}
}
func TestGeminiConfiguration(t *testing.T) {
	for _, model := range []string{"", "../other", "test?key=secret", "models/test"} {
		if _, e := NewGemini("secret", model); e == nil {
			t.Fatal("accepted unsafe model")
		}
	}
	if _, e := NewGemini("", "model"); e == nil {
		t.Fatal("accepted missing key")
	}
}
func TestGeminiCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := Gemini{APIKey: "secret", Model: "test-model", Client: &http.Client{Transport: geminiTransport(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })}}
	if _, e := p.Generate(ctx, domain.Input{ExternalConsent: true}); e == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestGeminiQuestionFocus(t *testing.T) {
	p := Gemini{APIKey: "test", Model: "test-model"}
	p.Client = &http.Client{Transport: geminiTransport(func(r *http.Request) (*http.Response, error) {
		var request struct {
			Contents []struct{ Parts []struct{ Text string } }
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal([]byte(request.Contents[0].Parts[0].Text), &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload) != 2 || string(payload["topic"]) != `"housing"` || string(payload["self_ratings"]) != `{"expenses":1}` {
			t.Fatal("unexpected data sent to external provider")
		}
		body := `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"focus\":\"resources\"}"}]}}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	focus, err := p.Focus(context.Background(), "housing", map[string]int32{"expenses": 1})
	if err != nil || focus != "resources" {
		t.Fatal("invalid focus response", err)
	}
}
