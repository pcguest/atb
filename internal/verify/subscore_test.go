// SPDX-License-Identifier: MIT
package verify

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestBuiltInProfileCASRegistryAlignment(t *testing.T) {
	tests := []struct {
		profileID string
		wantCAS   bool
		build     func(testing.TB) *bundle.Bundle
	}{
		{
			profileID: profileIDPrivilegedToolAction,
			wantCAS:   true,
			build:     newPrivilegedToolActionBundle,
		},
		{
			profileID: profileIDRAGAnswer,
			wantCAS:   true,
			build:     newRAGAnswerBundle,
		},
		{
			profileID: profileIDDataExport,
			wantCAS:   true,
			build:     newDataExportBundle,
		},
		{
			profileID: profileIDPolicyDecision,
			wantCAS:   true,
			build:     newPolicyDecisionBundle,
		},
		{
			profileID: profileIDHumanOverride,
			wantCAS:   true,
			build:     newHumanOverrideBundle,
		},
		{
			profileID: profileIDBackgroundAutomation,
			wantCAS:   true,
			build:     newBackgroundAutomationBundle,
		},
	}

	seen := map[string]struct{}{}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.profileID, func(t *testing.T) {
			if _, ok := seen[tc.profileID]; ok {
				t.Fatalf("duplicate test coverage for profile %q", tc.profileID)
			}
			seen[tc.profileID] = struct{}{}

			builder, hasBuilder := profileSubScoreBuilders[tc.profileID]
			if tc.wantCAS {
				if !SupportsCAS(tc.profileID) {
					t.Fatalf("expected %q to support CAS", tc.profileID)
				}
				if !hasBuilder || builder == nil {
					t.Fatalf("expected CAS-supported profile %q to have a sub-score builder", tc.profileID)
				}
			} else if SupportsCAS(tc.profileID) {
				t.Fatalf("did not expect %q to support CAS", tc.profileID)
			}

			report := Verify(tc.build(t), "bundle.atb", tc.profileID)
			if tc.wantCAS && report.CAS == nil {
				t.Fatalf("expected explicit profile %q to emit CAS", tc.profileID)
			}
			if !tc.wantCAS && report.CAS != nil {
				t.Fatalf("expected explicit profile %q to omit CAS, got %+v", tc.profileID, report.CAS)
			}
		})
	}

	if got, want := len(seen), len(AllProfiles()); got != want {
		t.Fatalf("covered %d built-in profiles, registry has %d", got, want)
	}
}

func TestSubScoresForBundle_CASProfilesRemainAvailable(t *testing.T) {
	tests := []struct {
		profileID string
		build     func(testing.TB) *bundle.Bundle
		wantSC    float64
	}{
		{profileID: profileIDDataExport, build: newDataExportBundle, wantSC: 0.60},
		{profileID: profileIDPolicyDecision, build: newPolicyDecisionBundle, wantSC: 0.60},
		{profileID: profileIDHumanOverride, build: newHumanOverrideBundle, wantSC: 0.45},
		{profileID: profileIDBackgroundAutomation, build: newBackgroundAutomationBundle, wantSC: 0.25},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.profileID, func(t *testing.T) {
			subScores := SubScoresForBundle(tc.build(t), "bundle.atb", tc.profileID)
			if allZeroSubScores(subScores) {
				t.Fatalf("expected retained sub-score builder for %q to produce non-zero output", tc.profileID)
			}
			if subScores["SC"] != tc.wantSC {
				t.Fatalf("expected SC=%.2f for %q, got %.3f", tc.wantSC, tc.profileID, subScores["SC"])
			}
		})
	}
}

func allZeroSubScores(subScores map[string]float64) bool {
	for _, value := range subScores {
		if value != 0 {
			return false
		}
	}
	return true
}

func newDataExportBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-data-export",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "data_export",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"export_allowed"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}, "2026-03-27T12:01:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-1",
		"action_type":              "export_data",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "dataset-1",
		"intended_effect":          "export approved dataset",
	}, "2026-03-27T12:02:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionExecuted, map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-03-27T12:03:00Z")
	appendVerifyRecord(t, b, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "appr-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approved",
		"justification_digest": "just-digest",
		"action_id":            "act-1",
	}, "2026-03-27T12:04:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionCommitted, map[string]any{
		"action_id":           "act-1",
		"commit_outcome":      "success",
		"sink_receipt_digest": "sink-digest",
	}, "2026-03-27T12:05:00Z")
	return b
}

func newPolicyDecisionBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-policy-decision",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "policy_decision",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"approved"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}, "2026-03-27T12:01:00Z")
	return b
}

func newHumanOverrideBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-human-override",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "human_override",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "appr-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approved",
		"justification_digest": "just-digest",
		"action_id":            "act-1",
	}, "2026-03-27T12:01:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-1",
		"action_type":              "override_action",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "run approved override",
	}, "2026-03-27T12:02:00Z")
	appendVerifyRecord(t, b, event.TypeAIActionExecuted, map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-03-27T12:03:00Z")
	return b
}

func newBackgroundAutomationBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIJobScheduled, map[string]any{
		"job_id":               "job-1",
		"job_type":             "nightly_sync",
		"trigger_source":       "cron",
		"scheduled_by_id_hash": "scheduler-hash",
	}, "2026-03-27T12:01:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobStarted, map[string]any{
		"job_id":         "job-1",
		"worker_id_hash": "worker-hash",
		"started_at":     "2026-03-27T12:02:00Z",
	}, "2026-03-27T12:02:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobStep, map[string]any{
		"job_id":       "job-1",
		"step_index":   1,
		"step_type":    "fetch_inputs",
		"step_outcome": "success",
	}, "2026-03-27T12:03:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobCompleted, map[string]any{
		"job_id":            "job-1",
		"outcome":           "success",
		"completion_reason": "completed",
	}, "2026-03-27T12:04:00Z")
	return b
}
