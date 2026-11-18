package domain

import (
	"errors"
	"slices"
)

var ErrNotFound = errors.New("assessment unavailable")
var ErrInvalid = errors.New("invalid result")

type SourceKind string

const SourceSurvey SourceKind = "survey"

type Signal struct {
	Ratings              map[string]int32 `json:"ratings,omitempty"`
	Summary              string           `json:"summary,omitempty"`
	Needs                []string         `json:"needs,omitempty"`
	Type                 SourceKind       `json:"type"`
	SourceVersion        int32            `json:"source_version"`
	ImmediateThreat      bool             `json:"immediate_threat"`
	InformationRequested bool             `json:"information_requested"`
	ExternalConsent      bool             `json:"external_consent"`
}
type Result struct {
	Version         int32            `json:"version,omitempty"`
	Ratings         map[string]int32 `json:"ratings,omitempty"`
	Summary         string           `json:"summary,omitempty"`
	Directions      []string         `json:"directions,omitempty"`
	Percentage      *int32           `json:"percentage,omitempty"`
	ID              string           `json:"id"`
	Status          string           `json:"status"`
	ImmediateThreat bool             `json:"immediate_threat_reported"`
	Recommendations []string         `json:"recommendations"`
	Disclaimer      string           `json:"disclaimer"`
	Provider        string           `json:"provider"`
}

const Disclaimer = "Демонстрационный результат, не диагноз и не валидированная оценка риска. Сервис не вызывает экстренную помощь и не является кризисным центром."

var Recommendations = map[string]string{"review_support_information": "Изучите доступную информацию о поддержке в удобное время.", "consider_trusted_contact": "Если это безопасно, рассмотрите обращение к человеку, которому доверяете.", "choose_safe_device": "При необходимости используйте устройство, к которому у посторонних нет доступа."}

func Validate(codes []string) error {
	if len(codes) < 1 || len(codes) > 3 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, v := range codes {
		if _, ok := Recommendations[v]; !ok || seen[v] {
			return ErrInvalid
		}
		seen[v] = true
	}
	return nil
}
func Build(id, status string, input Signal, codes []string, provider string) Result {
	r := Result{ID: id, Status: status, ImmediateThreat: input.ImmediateThreat, Disclaimer: Disclaimer, Provider: provider, Recommendations: []string{}}
	if input.SourceVersion == 2 {
		r.Version = 2
		r.Disclaimer = "Самооценка жизненной ситуации за неделю. Оценки по темам подтверждены вами."
		r.Summary = input.Summary
		r.Ratings = input.Ratings
		if len(input.Ratings) == 5 {
			var sum int32
			for _, v := range input.Ratings {
				sum += v
			}
			pct := 5 * sum
			r.Percentage = &pct
		}
		r.Directions = append(r.Directions, input.Needs...)
		// Finance requires a separately confirmed request. Other practical
		// resources can be suggested from a low rating in the matching area.
		for _, match := range []struct{ topic, direction string }{{"housing", "housing"}, {"work", "jobs"}, {"connection", "social"}} {
			if rating, ok := input.Ratings[match.topic]; ok && rating <= 1 && !slices.Contains(r.Directions, match.direction) {
				r.Directions = append(r.Directions, match.direction)
			}
		}
		if input.ImmediateThreat {
			r.Directions = append([]string{"emergency"}, r.Directions...)
		}
		for _, code := range codes {
			r.Recommendations = append(r.Recommendations, Recommendations[code])
		}
		return r
	}
	if input.ImmediateThreat {
		r.Recommendations = append(r.Recommendations, "В демонстрационном ответе явно сообщена непосредственная угроза. Если это соответствует реальной ситуации, по возможности перейдите в безопасное место и самостоятельно обратитесь в местную экстренную службу, когда это безопасно.")
	}
	for _, c := range codes {
		r.Recommendations = append(r.Recommendations, Recommendations[c])
	}
	return r
}
