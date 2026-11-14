package domain

import "errors"

var ErrInvalid = errors.New("invalid submission")
var ErrNotFound = errors.New("submission unavailable")

type Submission struct {
	UserID          string          `json:"user_id"`
	QuestionnaireID string          `json:"questionnaire_id"`
	Version         int32           `json:"version"`
	Answers         map[string]bool `json:"answers"`
	Consent         bool            `json:"consent"`
	ExternalConsent bool            `json:"external_llm_consent"`
	IdempotencyKey  string          `json:"idempotency_key"`
}
