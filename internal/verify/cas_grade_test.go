// SPDX-License-Identifier: MIT
package verify

import "testing"

func TestGradeFromScore_Boundaries(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{score: 1.0, want: "High"},
		{score: 0.85, want: "High"},
		{score: 0.849999, want: "Medium"},
		{score: 0.60, want: "Medium"},
		{score: 0.599999, want: "Low"},
		{score: 0.30, want: "Low"},
		{score: 0.299999, want: "Insufficient"},
		{score: 0.0, want: "Insufficient"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("", func(t *testing.T) {
			if got := gradeFromScore(tc.score); got != tc.want {
				t.Fatalf("gradeFromScore(%f) = %q, want %q", tc.score, got, tc.want)
			}
		})
	}
}

func TestComputeCAS_IntegrityFailureForcesInsufficient(t *testing.T) {
	subScores := map[string]float64{
		"EC": 1, "FC": 1, "RC": 1, "TC": 1, "SC": 1, "XC": 1, "AC": 1, "GC": 1,
	}
	weights := map[string]float64{
		"EC": 0.25, "FC": 0.15, "RC": 0.15, "TC": 0.10, "SC": 0.15, "XC": 0.05, "AC": 0.05, "GC": 0.10,
	}

	result := ComputeCAS(subScores, weights, false)
	if result.Overall != 0 {
		t.Fatalf("Overall = %f, want 0", result.Overall)
	}
	if result.Grade != "Insufficient" {
		t.Fatalf("Grade = %q, want Insufficient", result.Grade)
	}
}
