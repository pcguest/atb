// SPDX-License-Identifier: MIT
package verify

import (
	"slices"
	"testing"
)

func TestRequiredAnchorRisk(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)
	for _, required := range []bool{false, true} {
		report, err := EvaluateBundle(EvaluateConfig{Records: b.Records, AllApplicable: true, AnchorRequired: required})
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(report.ResidualRisk.Drivers, "required_anchor_unverified") != required {
			t.Fatalf("anchor driver: %+v", report.ResidualRisk)
		}
		if required && (report.ResidualRisk.Level != "High" || report.CAS.AssuranceValid) {
			t.Fatalf("required anchor failure not reflected: %+v", report)
		}
	}
}

func TestRequiredAnchorDoesNotDowngradeIntegrityFailure(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)
	b.Records[1].Hash = "tampered"
	report, err := EvaluateBundle(EvaluateConfig{Records: b.Records, AllApplicable: true, AnchorRequired: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.ResidualRisk.Level != "Critical" {
		t.Fatalf("risk=%+v", report.ResidualRisk)
	}
	if len(report.ResidualRisk.RecommendedNextEvidence) != 0 {
		t.Fatalf("critical integrity failure must not recommend timestamping: %+v", report.ResidualRisk)
	}
	if slices.Contains(report.ResidualRisk.Drivers, "required_anchor_unverified") {
		t.Fatalf("critical integrity failure must remain the primary driver: %+v", report.ResidualRisk)
	}
	if len(report.ProvabilityGaps) != 1 || report.ProvabilityGaps[0].Gap != "integrity" {
		t.Fatalf("critical integrity failure gaps = %+v, want only integrity", report.ProvabilityGaps)
	}
}

func TestRequiredAnchorProducesGapWithoutCAS(t *testing.T) {
	report := Report{
		Integrity:    IntegrityResult{ChainValid: true},
		Anchoring:    AnchoringResult{AnchorRequired: true},
		ResidualRisk: residualRiskNoMatchingProfile(),
	}
	applyRequiredAnchorRisk(&report)
	report.ProvabilityGaps = DeriveProvabilityGaps(report)
	if !slices.Contains(report.ResidualRisk.Drivers, "required_anchor_unverified") {
		t.Fatalf("required-anchor driver missing: %+v", report.ResidualRisk)
	}
	if !slices.ContainsFunc(report.ProvabilityGaps, func(gap ProvabilityGap) bool { return gap.Gap == "timestamp_anchor" }) {
		t.Fatalf("required-anchor provability gap missing: %+v", report.ProvabilityGaps)
	}
}
