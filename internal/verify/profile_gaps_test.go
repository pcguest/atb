// SPDX-License-Identifier: MIT
package verify_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// TestProfileGapsOnIntactButIncompleteBundle is the completeness counterpart to
// TestTamperDetection. Tampering and incompleteness are two different failure
// modes ATB deliberately keeps distinct: a tampered bundle fails *integrity*
// (the recorded bytes were altered — hash/chain failures), while an intact but
// incomplete bundle fails *provability* (the bytes are genuine, but the workflow
// skipped a control the obligation profile requires). This smoke drives the same
// real verifier path the viewer's ProfileCAS panel and `atb verify` use, and
// proves the second mode surfaces as obligation gaps rather than an integrity
// failure — and that completeness is scored, not binary.
func TestProfileGapsOnIntactButIncompleteBundle(t *testing.T) {
	t.Parallel()

	// Baseline: the complete, intact bundle has no critical failures and a
	// positive CAS score (the same expectation as TestTamperDetection's valid
	// case), so it is a sound reference to contrast the incomplete bundle against.
	completePath := writePrivilegedToolActionBundle(t)
	completeReport, err := evaluateBundle(completePath)
	if err != nil {
		t.Fatalf("evaluate complete bundle: %v", err)
	}
	if len(completeReport.Failures) != 0 {
		t.Fatalf("complete bundle critical failures = %v, want none", completeReport.Failures)
	}
	if completeReport.CASScore <= 0 {
		t.Fatalf("complete bundle CAS score = %f, want > 0", completeReport.CASScore)
	}

	// The bundle under test is byte-intact (valid hash chain) but omits the
	// oversight obligations the privileged_tool_action profile requires — a
	// privileged action that ran with no policy decision and no human approval.
	incompletePath := writeIncompletePrivilegedToolActionBundle(t)
	incompleteReport, err := evaluateBundle(incompletePath)
	if err != nil {
		t.Fatalf("evaluate incomplete bundle: %v", err)
	}

	// 1. This is a completeness gap, not tampering: the hash chain is intact, so
	//    none of the reported failures are integrity (tamper/hash/chain) failures.
	for _, failure := range incompleteReport.Failures {
		kind := strings.ToLower(failure.Kind)
		if strings.Contains(kind, "tamper") || strings.Contains(kind, "hash") || strings.Contains(kind, "chain") {
			t.Fatalf("intact-but-incomplete bundle reported an integrity failure %q; expected only obligation gaps", failure.Kind)
		}
	}

	// 2. The profile gate fails and the missing obligations surface — as critical
	//    failures and/or provability gaps (the ProfileCAS panel reads both).
	if incompleteReport.Pass {
		t.Fatalf("incomplete bundle unexpectedly passed its profile gate")
	}
	if len(incompleteReport.Failures) == 0 && len(incompleteReport.ProvabilityGaps) == 0 {
		t.Fatalf("incomplete bundle surfaced no obligation failures or provability gaps")
	}

	// 3. Completeness is scored, not binary: the intact-but-incomplete bundle
	//    scores strictly below the complete one.
	if incompleteReport.CASScore >= completeReport.CASScore {
		t.Fatalf("incomplete CAS score %.3f not below complete CAS score %.3f",
			incompleteReport.CASScore, completeReport.CASScore)
	}
}

// writeIncompletePrivilegedToolActionBundle builds a byte-valid bundle for the
// privileged_tool_action profile that deliberately omits the policy-decision and
// human-approval oversight events, so a privileged action appears to have run
// with no recorded authorisation. It reuses appendRecord from tamper_test.go.
func writeIncompletePrivilegedToolActionBundle(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	bundlePath := filepath.Join(root, bundle.BundleDir, bundle.BundleFile)

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	appendRecord(t, bundlePath, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "privileged_tool_action",
	}, "2026-03-27T12:00:00Z")
	appendRecord(t, bundlePath, event.TypeAILLMCall, map[string]any{
		"call_id":    "call-1",
		"model_id":   "gpt-5.4",
		"request_id": "req-1",
	}, "2026-03-27T12:00:30Z")
	appendRecord(t, bundlePath, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-1",
		"action_type":              "deploy_change",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "deploy build 42",
	}, "2026-03-27T12:01:00Z")
	// Tool execution and the executed action are present, but no policy decision
	// and no human approval precede them.
	appendRecord(t, bundlePath, event.TypeAIToolExec, map[string]any{
		"action_id":  "act-1",
		"tool_name":  "deployctl",
		"exit_code":  0,
		"request_id": "req-1",
	}, "2026-03-27T12:02:30Z")
	appendRecord(t, bundlePath, event.TypeAIActionExecuted, map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-03-27T12:03:00Z")

	return bundlePath
}
