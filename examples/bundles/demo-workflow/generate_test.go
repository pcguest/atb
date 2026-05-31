// SPDX-License-Identifier: MIT
package main

import (
	"testing"

	verifypkg "github.com/pcguest/atb/internal/verify"
)

// TestBuildDemoBundleDefaultProfilePasses pins the documented demo command:
//
//	atb verify --bundle examples/bundles/demo-workflow/demo-workflow.atb
//
// With no --profile, verify must auto-select the bundle's declared workflow
// class (policy_decision, set via purpose_tag on ai.request.received) and
// report PASS. This guards against the profile-selection trap where, absent a
// purpose tag, the heuristic fell through to privileged_tool_action and
// reported FAIL — making a reviewer think the demo was broken.
//
// It runs against the in-process builder rather than the committed .atb so it
// is reproducible in CI, where the bundle artefact is gitignored.
func TestBuildDemoBundleDefaultProfilePasses(t *testing.T) {
	b, err := buildDemoBundle()
	if err != nil {
		t.Fatalf("build demo bundle: %v", err)
	}

	report, err := verifypkg.EvaluateBundle(verifypkg.EvaluateConfig{
		BundlePath:    "demo-workflow.atb",
		Records:       b.Records,
		AllApplicable: true, // mirrors `atb verify` with no --profile
	})
	if err != nil {
		t.Fatalf("evaluate demo bundle: %v", err)
	}

	if !report.Integrity.ChainValid {
		t.Fatal("integrity: want chain valid, got invalid")
	}
	if len(report.Profiles) != 1 {
		t.Fatalf("default profile selection: want exactly 1 profile, got %d", len(report.Profiles))
	}
	got := report.Profiles[0]
	if got.ProfileID != "atb.profile.policy_decision" {
		t.Fatalf("default profile: want atb.profile.policy_decision, got %q", got.ProfileID)
	}
	if !got.Pass {
		t.Fatalf("default profile policy_decision: want PASS, got FAIL: %+v", got.CriticalFailures)
	}
}
