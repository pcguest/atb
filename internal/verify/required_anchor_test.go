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
}
