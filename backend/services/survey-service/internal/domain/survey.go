package domain

import "errors"

var ErrInvalid = errors.New("invalid submission")
var ErrNotFound = errors.New("submission unavailable")

type Submission struct {
	QuestionFocus   map[string]string `json:"question_focus,omitempty"`
	Responses       map[string]string `json:"responses,omitempty"`
	Ratings         map[string]int32  `json:"ratings,omitempty"`
	Summary         string            `json:"summary,omitempty"`
	Needs           []string          `json:"needs,omitempty"`
	UserID          string            `json:"user_id"`
	QuestionnaireID string            `json:"questionnaire_id"`
	Version         int32             `json:"version"`
	Answers         map[string]bool   `json:"answers"`
	Consent         bool              `json:"consent"`
	ExternalConsent bool              `json:"external_llm_consent"`
	IdempotencyKey  string            `json:"idempotency_key"`
}
