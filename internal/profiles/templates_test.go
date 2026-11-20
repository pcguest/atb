// SPDX-License-Identifier: MIT
package profiles

import (
	"testing"

	"github.com/pcguest/atb/v2/internal/bundle"
	"github.com/pcguest/atb/v2/internal/event"
)

func TestProfile_PrivilegedToolAction_RequiredWhen(t *testing.T) {
	schema := MustLoadSchema("atb.profile.privileged_tool_action")

	buildWithExecutions := func(executionTimestamps []string, approvals ...bundle.Record) []bundle.Record {
		records := []bundle.Record{
			record(event.TypeBundleManifest, nil),
			record(event.TypeAIRequestReceived, map[string]any{"request_id": "r1", "actor_id_hash": "a1", "purpose_tag": "p1"}),
			record(event.TypeAIActionPrecommit, map[string]any{"action_id": "act1", "action_type": "shell", "action_parameters_digest": "d1", "target_resource_id": "res1", "intended_effect": "e1"}),
			record(event.TypeAIPolicyDecision, map[string]any{"policy_id": "pol1", "policy_version": "v1", "decision": "allow", "decision_reason_codes": []string{"ok"}, "subject_id_hash": "s1", "action_id": "act1"}),
		}
		for _, executionTimestamp := range executionTimestamps {
			records = append(records, recordWithTimestamp(event.TypeAIActionExecuted, map[string]any{"action_id": "act1", "execution_outcome": "success", "tool_receipt_digest": "tr1"}, executionTimestamp))
		}
		records = append(records, record(event.TypeAIActionCommitted, map[string]any{"action_id": "act1", "commit_outcome": "success", "sink_receipt_digest": "sr1"}))
		return append(records, approvals...)
	}
	build := func(executionTimestamp string, approvals ...bundle.Record) []bundle.Record {
		return buildWithExecutions([]string{executionTimestamp}, approvals...)
	}
	approval := func(actionID, outcome, timestamp string, approvalIDs ...string) bundle.Record {
		approvalID := "app1"
		if len(approvalIDs) > 0 {
			approvalID = approvalIDs[0]
		}
		return recordWithTimestamp(event.TypeAIHumanApproval, map[string]any{"approval_id": approvalID, "approver_id_hash": "h1", "approval_outcome": outcome, "justification_digest": "j1", "action_id": actionID}, timestamp)
	}

	t.Run("approval before execution passes", func(t *testing.T) {
		result := Evaluate(schema, build("2026-03-27T12:03:00Z", approval("act1", "approved", "2026-03-27T12:02:30Z")))
		if !result.Pass {
			t.Fatalf("expected pass, got failures: %+v", result.CriticalFailures)
		}
	})

	t.Run("each execution with the same action_id needs a distinct approval", func(t *testing.T) {
		oneApproval := buildWithExecutions(
			[]string{"2026-03-27T12:03:00Z", "2026-03-27T12:04:00Z"},
			approval("act1", "approved", "2026-03-27T12:02:30Z"),
		)
		result := Evaluate(schema, oneApproval)
		if result.Pass {
			t.Fatal("expected failure due to reused approval")
		}
		if !hasFailure(result.CriticalFailures, "reused_correlated_event", "action_id=act1") {
			t.Fatalf("expected reused_correlated_event, got %+v", result.CriticalFailures)
		}

		twoApprovals := buildWithExecutions(
			[]string{"2026-03-27T12:03:00Z", "2026-03-27T12:04:00Z"},
			approval("act1", "approved", "2026-03-27T12:02:30Z", "app1"),
			approval("act1", "approved", "2026-03-27T12:03:30Z", "app2"),
		)
		result = Evaluate(schema, twoApprovals)
		if !result.Pass {
			t.Fatalf("expected pass with distinct approvals, got failures: %+v", result.CriticalFailures)
		}
	})

	t.Run("missing approval fails", func(t *testing.T) {
		result := Evaluate(schema, build("2026-03-27T12:03:00Z"))
		if result.Pass {
			t.Fatal("expected failure due to missing ai.human.approval")
		}
		if !hasFailure(result.CriticalFailures, "missing_event", "ai.human.approval must precede ai.action.executed for the same action_id") {
			t.Fatalf("expected missing_event for ai.human.approval, got %+v", result.CriticalFailures)
		}
	})

	t.Run("approval after execution fails", func(t *testing.T) {
		result := Evaluate(schema, build("2026-03-27T12:03:00Z", approval("act1", "approved", "2026-03-27T12:03:30Z")))
		if result.Pass {
			t.Fatal("expected failure due to temporal violation")
		}
		if !hasFailure(result.CriticalFailures, "temporal_violation", "must occur before") {
			t.Fatalf("expected temporal_violation, got %+v", result.CriticalFailures)
		}
	})

	t.Run("approval for another action fails", func(t *testing.T) {
		result := Evaluate(schema, build("2026-03-27T12:03:00Z", approval("act2", "approved", "2026-03-27T12:02:30Z")))
		if result.Pass {
			t.Fatal("expected failure due to uncorrelated approval")
		}
		if !hasFailure(result.CriticalFailures, "missing_correlated_event", "ai.human.approval") {
			t.Fatalf("expected missing_correlated_event, got %+v", result.CriticalFailures)
		}
	})

	t.Run("missing approval timestamp fails", func(t *testing.T) {
		result := Evaluate(schema, build("2026-03-27T12:03:00Z", approval("act1", "approved", "")))
		if result.Pass {
			t.Fatal("expected failure due to unknown approval time")
		}
		if !hasFailure(result.CriticalFailures, "temporal_unknown", "ai.human.approval") {
			t.Fatalf("expected temporal_unknown for approval, got %+v", result.CriticalFailures)
		}
	})

	t.Run("missing execution timestamp fails", func(t *testing.T) {
		result := Evaluate(schema, build("", approval("act1", "approved", "2026-03-27T12:02:30Z")))
		if result.Pass {
			t.Fatal("expected failure due to unknown execution time")
		}
		if !hasFailure(result.CriticalFailures, "temporal_unknown", "ai.action.executed") {
			t.Fatalf("expected temporal_unknown for execution, got %+v", result.CriticalFailures)
		}
	})

	t.Run("denied approval fails relation", func(t *testing.T) {
		result := Evaluate(schema, build("2026-03-27T12:03:00Z", approval("act1", "denied", "2026-03-27T12:02:30Z")))
		if result.Pass {
			t.Fatal("expected failure due to denied approval")
		}
		if !hasFailure(result.CriticalFailures, "relation_violation", "approval_binds_action") {
			t.Fatalf("expected approval_binds_action relation failure, got %+v", result.CriticalFailures)
		}
	})
}
