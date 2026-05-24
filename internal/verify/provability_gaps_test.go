// SPDX-License-Identifier: MIT
package verify_test

import (
	"testing"

	"github.com/pcguest/atb/internal/verify"
)

func TestDeriveProvabilityGaps_IntegrityFailure(t *testing.T) {
	t.Parallel()

	gaps := verify.DeriveProvabilityGaps(verify.Report{
		Integrity: verify.IntegrityResult{ChainValid: false},
	})
	if len(gaps) != 1 || gaps[0].Layer != "L1" {
		t.Fatalf("gaps = %+v, want single L1 integrity gap", gaps)
	}
}

func TestDeriveProvabilityGaps_LowSC(t *testing.T) {
	t.Parallel()

	gaps := verify.DeriveProvabilityGaps(verify.Report{
		Integrity: verify.IntegrityResult{ChainValid: true},
		CAS: &verify.CASResult{
			SubScores: map[string]float64{"SC": 0.1},
		},
		Profiles: []verify.ProfileResult{{Pass: true}},
	})

	found := false
	for _, gap := range gaps {
		if gap.Gap == "source_binding" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected source_binding gap, got %+v", gaps)
	}
}
