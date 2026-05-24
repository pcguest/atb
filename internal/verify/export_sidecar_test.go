// SPDX-License-Identifier: MIT
package verify

import "testing"

func TestProvabilityLayerChecklist(t *testing.T) {
	report := Report{
		Integrity: IntegrityResult{ChainValid: true},
		Profiles:  []ProfileResult{{Pass: true}},
		CAS: &CASResult{
			SubScores: map[string]float64{
				"SC": 0.75,
				"XC": 1.0,
				"AC": 0.5,
			},
		},
		Anchoring:       AnchoringResult{TSAVerified: true},
		BundleSignature: &BundleSignatureResult{Verified: true},
		Retrospective:   false,
	}

	checklist := ProvabilityLayerChecklist(report)
	if !checklist["L1_integrity"] {
		t.Fatal("expected L1_integrity true")
	}
	if !checklist["L2_profile_pass"] {
		t.Fatal("expected L2_profile_pass true")
	}
	if !checklist["L3_source_binding"] {
		t.Fatal("expected L3_source_binding true")
	}
	if !checklist["L4_external_witness"] {
		t.Fatal("expected L4_external_witness true")
	}
	if !checklist["L4_anchor"] {
		t.Fatal("expected L4_anchor true")
	}
	if !checklist["L3_bundle_signature"] {
		t.Fatal("expected L3_bundle_signature true")
	}
	if !checklist["L5_live_capture"] {
		t.Fatal("expected L5_live_capture true for non-retrospective bundle")
	}
}
