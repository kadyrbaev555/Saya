package domain

import "testing"

func TestDeterministicRuleSurvivesFailure(t *testing.T) {
	r := Build("demo", "failed", Signal{Type: "survey", SourceVersion: 1, ImmediateThreat: true}, nil, "unavailable")
	if !r.ImmediateThreat || len(r.Recommendations) != 1 || r.Disclaimer == "" {
		t.Fatal("explicit threat rule lost during LLM failure")
	}
	if Validate([]string{"diagnosis"}) == nil || Validate([]string{"choose_safe_device", "choose_safe_device"}) == nil {
		t.Fatal("invalid structured output accepted")
	}
}
