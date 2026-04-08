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
			wantCAS:   false,
			build:     newDataExportBundle,
		},
		{
			profileID: profileIDPolicyDecision,
			wantCAS:   false,
			build:     newPolicyDecisionBundle,
		},
		{
			profileID: profileIDHumanOverride,
			wantCAS:   false,
			build:     newHumanOverrideBundle,
		},
		{
			profileID: profileIDBackgroundAutomation,
			wantCAS:   false,
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

func TestSubScoresForBundle_NonCASProfilesRemainAvailable(t *testing.T) {
	tests := []struct {
		profileID string
		build     func(testing.TB) *bundle.Bundle
	}{
		{profileID: profileIDDataExport, build: newDataExportBundle},
		{profileID: profileIDPolicyDecision, build: newPolicyDecisionBundle},
		{profileID: profileIDHumanOverride, build: newHumanOverrideBundle},
		{profileID: profileIDBackgroundAutomation, build: newBackgroundAutomationBundle},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.profileID, func(t *testing.T) {
			subScores := SubScoresForBundle(tc.build(t), "bundle.atb", tc.profileID)
			if allZeroSubScores(subScores) {
				t.Fatalf("expected retained sub-score builder for %q to produce non-zero output", tc.profileID)
			}
			if subScores["SC"] != 0 {
				t.Fatalf("expected SC=0 for non-CAS profile %q, got %.3f", tc.profileID, subScores["SC"])
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
		"job_id":       "job-1",
		"schedule_id":  "schedule-1",
		"job_type":     "nightly_sync",
		"trigger_type": "cron",
	}, "2026-03-27T12:01:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobStarted, map[string]any{
		"job_id":       "job-1",
		"worker_id":    "worker-1",
		"start_reason": "scheduled_trigger",
	}, "2026-03-27T12:02:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobStep, map[string]any{
		"job_id":       "job-1",
		"step_id":      "step-1",
		"step_name":    "fetch_inputs",
		"step_outcome": "success",
	}, "2026-03-27T12:03:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobCompleted, map[string]any{
		"job_id":             "job-1",
		"completion_outcome": "success",
		"result_digest":      "sink-digest",
	}, "2026-03-27T12:04:00Z")
	return b
}
