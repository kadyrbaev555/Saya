package domain

import "errors"

type Input struct {
	Version              int32            `json:"-"`
	Ratings              map[string]int32 `json:"ratings,omitempty"`
	Needs                []string         `json:"needs,omitempty"`
	ImmediateThreat      bool             `json:"immediate_threat"`
	InformationRequested bool             `json:"information_requested"`
	ExternalConsent      bool             `json:"-"`
}
type Output struct {
	Recommendations []string `json:"recommendations"`
}

func (o Output) Validate() error {
	if len(o.Recommendations) < 1 || len(o.Recommendations) > 3 {
		return errors.New("invalid structured response")
	}
	seen := map[string]bool{}
	for _, code := range o.Recommendations {
		if (code != "review_support_information" && code != "consider_trusted_contact" && code != "choose_safe_device") || seen[code] {
			return errors.New("invalid recommendation code")
		}
		seen[code] = true
	}
	return nil
}
