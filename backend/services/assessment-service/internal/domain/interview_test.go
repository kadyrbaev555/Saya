package domain

import (
	"slices"
	"testing"
)

func TestInterviewAssessment(t *testing.T) {
	in := Signal{SourceVersion: 2, Ratings: map[string]int32{"housing": 1, "expenses": 1, "work": 2, "connection": 3, "daily": 3}, Summary: "Confirmed", Needs: []string{"psychologist"}, ImmediateThreat: true}
	r := Build("id", "completed", in, nil, "self-report")
	if r.Percentage == nil || *r.Percentage != 50 || r.Summary != "Confirmed" || !slices.Equal(r.Directions, []string{"emergency", "psychologist", "housing"}) {
		t.Fatalf("unexpected result: %+v", r)
	}
	delete(in.Ratings, "daily")
	if Build("id", "completed", in, nil, "self-report").Percentage != nil {
		t.Fatal("partial scale acquired a percentage")
	}
}
