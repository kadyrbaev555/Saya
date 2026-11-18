package domain

import "testing"

func TestInterviewValidation(t *testing.T) {
	valid := func() Submission {
		return Submission{QuestionnaireID: "saya-week", Version: 2, Consent: true, Ratings: map[string]int32{"expenses": 0}, Needs: []string{"financial"}}
	}
	if !valid().ValidInterview() {
		t.Fatal("zero rating must be valid")
	}
	for _, change := range []func(*Submission){func(s *Submission) { s.Consent = false }, func(s *Submission) { delete(s.Ratings, "expenses") }, func(s *Submission) { s.Ratings["expenses"] = 2 }, func(s *Submission) { s.Ratings["priority"] = 0 }, func(s *Submission) { s.Needs = append(s.Needs, "financial") }, func(s *Submission) { s.Responses = map[string]string{"unknown": "x"} }} {
		s := valid()
		change(&s)
		if s.ValidInterview() {
			t.Fatal("accepted invalid interview")
		}
	}
}
