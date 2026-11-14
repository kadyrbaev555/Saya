package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"saya/ai-service/internal/domain"
	"strings"
	"time"
)

// Gemini uses Google's REST API. Keys are sent only in the request header.
type Gemini struct {
	APIKey, Model string
	Client        *http.Client
}

var modelID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]{0,127}$`)

func NewGemini(key, model string) (Gemini, error) {
	if strings.TrimSpace(key) == "" || !modelID.MatchString(model) {
		return Gemini{}, errors.New("GEMINI_API_KEY and valid GEMINI_MODEL required")
	}
	return Gemini{APIKey: key, Model: model}, nil
}
func (Gemini) Name() string { return "gemini" }
func (p Gemini) Generate(ctx context.Context, input domain.Input) (domain.Output, error) {
	var output domain.Output
	if !input.ExternalConsent {
		return output, errors.New("external processing consent required")
	}
	if _, e := NewGemini(p.APIKey, p.Model); e != nil {
		return output, e
	}
	signals, e := json.Marshal(input)
	if e != nil {
		return output, e
	}
	schema := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"recommendations"}, "properties": map[string]any{"recommendations": map[string]any{"type": "array", "minItems": 1, "maxItems": 3, "items": map[string]any{"type": "string", "enum": []string{"review_support_information", "consider_trusted_contact", "choose_safe_device"}}}}}
	body, e := json.Marshal(map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]string{{"text": "Select approved information-support recommendation codes from supplied voluntary survey signals. Do not diagnose, assign risk scores, claim certainty, or initiate emergency actions. Return only the JSON schema. Explicit threat rules run separately in the backend."}}},
		"contents":          []map[string]any{{"role": "user", "parts": []map[string]string{{"text": string(signals)}}}},
		"generationConfig":  map[string]any{"responseMimeType": "application/json", "responseJsonSchema": schema, "candidateCount": 1, "maxOutputTokens": 512},
	})
	if e != nil {
		return output, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, "https://generativelanguage.googleapis.com/v1beta/models/"+p.Model+":generateContent", bytes.NewReader(body))
	if e != nil {
		return output, e
	}
	req.Header.Set("x-goog-api-key", p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, e := client.Do(req)
	if e != nil {
		return output, errors.New("Gemini transport failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return output, errors.New("Gemini request rejected")
	}
	data, e := io.ReadAll(io.LimitReader(response.Body, 65537))
	if e != nil || len(data) > 65536 {
		return output, errors.New("Gemini response exceeds limit")
	}
	var envelope struct {
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.PromptFeedback.BlockReason != "" || len(envelope.Candidates) != 1 || envelope.Candidates[0].FinishReason != "STOP" {
		return output, errors.New("Gemini output unavailable or incomplete")
	}
	var text strings.Builder
	for _, part := range envelope.Candidates[0].Content.Parts {
		if !part.Thought {
			text.WriteString(part.Text)
		}
	}
	if text.Len() > 16384 {
		return output, errors.New("Gemini structured output exceeds limit")
	}
	decoder := json.NewDecoder(strings.NewReader(text.String()))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&output) != nil {
		return output, errors.New("invalid Gemini structured output")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return output, errors.New("trailing Gemini output")
	}
	return output, output.Validate()
}
