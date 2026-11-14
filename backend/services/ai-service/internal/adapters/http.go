package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"saya/ai-service/internal/domain"
	"time"
)

// HTTPProvider calls a configured structured LLM endpoint; see docs/api/llm.md.
type HTTPProvider struct {
	Endpoint, APIKey string
	Client           *http.Client
}

func (p HTTPProvider) Name() string { return "external-http" }
func (p HTTPProvider) Generate(ctx context.Context, input domain.Input) (domain.Output, error) {
	var output domain.Output
	if !input.ExternalConsent {
		return output, errors.New("external processing consent required")
	}
	endpoint, e := url.Parse(p.Endpoint)
	if e != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil {
		return output, errors.New("LLM endpoint must be HTTPS")
	}
	body, e := json.Marshal(map[string]any{"version": 1, "demonstration": true, "input": input, "allowed_recommendations": []string{"review_support_information", "consider_trusted_contact", "choose_safe_device"}, "instructions": "Return only JSON recommendations using allowed codes. No diagnosis, risk score, or emergency actions."})
	if e != nil {
		return output, e
	}
	req, e := http.NewRequestWithContext(ctx, "POST", p.Endpoint, bytes.NewReader(body))
	if e != nil {
		return output, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, e := client.Do(req)
	if e != nil {
		return output, errors.New("LLM transport failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return output, errors.New("LLM response rejected")
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, 16385))
	if e != nil || len(data) > 16384 {
		return output, errors.New("LLM response too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&output); e != nil {
		return output, errors.New("invalid structured output")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return output, errors.New("trailing output")
	}
	return output, output.Validate()
}
