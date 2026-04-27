// SPDX-License-Identifier: MIT
package verify

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// newPrivilegedBundleNoAnchor builds a minimal privileged_tool_action bundle
// without an anchor event so the baseline XC is 0.0 (AnchorAbsent).
func newPrivilegedBundleNoAnchor(t testing.TB) *bundle.Bundle {
	t.Helper()
	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "approve-change",
	}, "2026-04-20T09:00:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-1",
		"action_type":              "deploy_change",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "deploy build 42",
	}, "2026-04-20T09:01:00Z")
	appendVerifyRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}, "2026-04-20T09:02:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionExecuted, map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-04-20T09:03:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionCommitted, map[string]any{
		"action_id":           "act-1",
		"commit_outcome":      "success",
		"sink_receipt_digest": "sink-digest",
	}, "2026-04-20T09:04:00Z")
	return b
}

// TestXCCorroboration_EndToEnd verifies that a bundle containing a valid
// atb.corroboration.external event earns non-zero XC in the CAS output,
// and that a bundle without corroboration events and without an anchor
// scores XC=0.0 (preserving v1.9.0 behaviour for anchor-absent bundles).
func TestXCCorroboration_EndToEnd(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	// Baseline: no corroboration, no anchor → XC must be 0.0.
	baseNoCorr := newPrivilegedBundleNoAnchor(t)
	bundlePathNoCorr := writeEvaluateBundleFixture(t, baseNoCorr)
	reportNoCorr, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePathNoCorr,
		Profiles:   []Profile{profile},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle (no corroboration): %v", err)
	}
	if reportNoCorr.CAS == nil {
		t.Fatal("expected CAS in no-corroboration report")
	}
	xcNoCorr := reportNoCorr.CAS.SubScores["XC"]
	if xcNoCorr != 0.0 {
		t.Errorf("expected XC=0.0 for bundle with no corroboration and no anchor, got %.3f", xcNoCorr)
	}

	// Add a valid corroboration event to an otherwise identical bundle.
	baseWithCorr := newPrivilegedBundleNoAnchor(t)
	refHash := baseWithCorr.Records[1].Hash
	appendVerifyRecord(t, baseWithCorr, event.TypeCorroborationExternal, map[string]any{
		"source":       "http-gateway",
		"reference_id": refHash,
		"digest":       "a" + refHash[1:],
		"retrieved_at": "2026-04-20T10:00:00Z",
		"adapter":      "http-gateway",
	}, "2026-04-20T10:00:00Z")

	bundlePath := writeEvaluateBundleFixture(t, baseWithCorr)
	report, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	if report.CAS == nil {
		t.Fatal("expected CAS in report")
	}
	xc := report.CAS.SubScores["XC"]
	if xc <= 0.0 {
		t.Errorf("expected XC > 0.0 for bundle with corroboration event, got %.3f", xc)
	}

	// Regression: all non-XC sub-scores must be identical between the two bundles.
	for dim, wantScore := range reportNoCorr.CAS.SubScores {
		if dim == "XC" {
			continue
		}
		gotScore := report.CAS.SubScores[dim]
		if gotScore != wantScore {
			t.Errorf("sub-score %s changed unexpectedly: got %.3f, want %.3f", dim, gotScore, wantScore)
		}
	}
}

// TestXCCorroboration_WithAnchorBundle verifies that adding a corroboration
// event to a bundle that already has an anchor does not decrease XC.
func TestXCCorroboration_WithAnchorBundle(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	// The standard fixture includes an anchor with AnchorPresentBadData (XC=0.1).
	baseNoCorr := newPrivilegedToolActionBundle(t)
	bundlePathNoCorr := writeEvaluateBundleFixture(t, baseNoCorr)
	reportNoCorr, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePathNoCorr,
		Profiles:   []Profile{profile},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle (no corroboration): %v", err)
	}
	xcNoCorr := reportNoCorr.CAS.SubScores["XC"]

	// Add a valid corroboration event to an identical bundle.
	baseWithCorr := newPrivilegedToolActionBundle(t)
	refHash := baseWithCorr.Records[1].Hash
	appendVerifyRecord(t, baseWithCorr, event.TypeCorroborationExternal, map[string]any{
		"source":       "sqs",
		"reference_id": refHash,
		"digest":       "deadbeefdeadbeef",
		"retrieved_at": "2026-04-20T10:00:00Z",
	}, "2026-04-20T10:00:00Z")

	bundlePath := writeEvaluateBundleFixture(t, baseWithCorr)
	reportWithCorr, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle (with corroboration): %v", err)
	}
	xcWithCorr := reportWithCorr.CAS.SubScores["XC"]

	if xcWithCorr < xcNoCorr {
		t.Errorf("XC decreased with corroboration: was %.3f, now %.3f", xcNoCorr, xcWithCorr)
	}
	if xcWithCorr <= 0.0 {
		t.Errorf("expected XC > 0.0 with corroboration event, got %.3f", xcWithCorr)
	}
}

// TestXCCorroboration_InvalidEventDoesNotScore verifies that an
// atb.corroboration.external event missing required fields does not
// contribute to XC.
func TestXCCorroboration_InvalidEventDoesNotScore(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	b := newPrivilegedBundleNoAnchor(t)
	appendVerifyRecord(t, b, event.TypeCorroborationExternal, map[string]any{
		"source": "http-gateway",
		// missing reference_id, digest, retrieved_at
	}, "2026-04-20T10:00:00Z")

	bundlePath := writeEvaluateBundleFixture(t, b)
	report, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	if report.CAS == nil {
		t.Fatal("expected CAS in report")
	}
	xc := report.CAS.SubScores["XC"]
	if xc != 0.0 {
		t.Errorf("expected XC=0.0 for invalid corroboration event (no anchor), got %.3f", xc)
	}
}

// TestXCCorroboration_BadTimestampDoesNotScore verifies that a corroboration
// event with an unparseable retrieved_at timestamp does not contribute to XC.
func TestXCCorroboration_BadTimestampDoesNotScore(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	b := newPrivilegedBundleNoAnchor(t)
	appendVerifyRecord(t, b, event.TypeCorroborationExternal, map[string]any{
		"source":       "http-gateway",
		"reference_id": "some-ref",
		"digest":       "abc123",
		"retrieved_at": "not-a-timestamp",
	}, "2026-04-20T10:00:00Z")

	bundlePath := writeEvaluateBundleFixture(t, b)
	report, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	xc := report.CAS.SubScores["XC"]
	if xc != 0.0 {
		t.Errorf("expected XC=0.0 for corroboration event with bad timestamp (no anchor), got %.3f", xc)
	}
}

// TestCountValidCorroborationEvents tests the counting helper directly.
func TestCountValidCorroborationEvents(t *testing.T) {
	b := newVerifyTestBundle(t)

	if got := countValidCorroborationEvents(b.Records); got != 0 {
		t.Errorf("expected 0 valid corroboration events in empty bundle, got %d", got)
	}

	appendVerifyRecord(t, b, event.TypeCorroborationExternal, map[string]any{
		"source":       "sqs",
		"reference_id": "ref-1",
		"digest":       "deadbeef",
		"retrieved_at": "2026-04-20T10:00:00Z",
	}, "2026-04-20T10:00:00Z")

	if got := countValidCorroborationEvents(b.Records); got != 1 {
		t.Errorf("expected 1 valid corroboration event, got %d", got)
	}

	appendVerifyRecord(t, b, event.TypeCorroborationExternal, map[string]any{
		"source":       "s3",
		"reference_id": "ref-2",
		"digest":       "cafebabe",
		"retrieved_at": "2026-04-20T10:01:00Z",
	}, "2026-04-20T10:01:00Z")

	if got := countValidCorroborationEvents(b.Records); got != 2 {
		t.Errorf("expected 2 valid corroboration events, got %d", got)
	}
}
