// SPDX-License-Identifier: MIT
package profiles

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestProfile_PrivilegedToolAction_RequiredWhen(t *testing.T) {
	schema := MustLoadSchema("atb.profile.privileged_tool_action")

	// 1. Setup minimal records that trigger required_when: ai.action.executed is present.
	records := []bundle.Record{
		record(event.TypeBundleManifest, nil),
		record(event.TypeAIRequestReceived, map[string]any{"request_id": "r1", "actor_id_hash": "a1", "purpose_tag": "p1"}),
		record(event.TypeAIActionPrecommit, map[string]any{"action_id": "act1", "action_type": "shell", "action_parameters_digest": "d1", "target_resource_id": "res1", "intended_effect": "e1"}),
		record(event.TypeAIPolicyDecision, map[string]any{"policy_id": "pol1", "policy_version": "v1", "decision": "allow", "decision_reason_codes": []string{"ok"}, "subject_id_hash": "s1", "action_id": "act1"}),
		recordWithTimestamp(event.TypeAIActionExecuted, map[string]any{"action_id": "act1", "execution_outcome": "success", "tool_receipt_digest": "tr1"}, "2026-03-27T12:00:00Z"),
		record(event.TypeAIActionCommitted, map[string]any{"action_id": "act1", "commit_outcome": "success", "sink_receipt_digest": "sr1"}),
	}

	// 2. Verify that missing the required human approval yields a failure.
	result := Evaluate(schema, records)
	if result.Pass {
		t.Fatal("expected failure due to missing ai.human.approval")
	}
	if !hasFailure(result.CriticalFailures, "missing_event", "ai.human.approval required when actions execute") {
		t.Fatalf("expected missing_event for ai.human.approval, got %+v", result.CriticalFailures)
	}

	// 3. Add the human approval event in the WRONG order (before execution).
	recordsWithWrongOrder := append([]bundle.Record(nil), records...)
	recordsWithWrongOrder = append(recordsWithWrongOrder, recordWithTimestamp(event.TypeAIHumanApproval, map[string]any{"approval_id": "app1", "approver_id_hash": "h1", "approval_outcome": "approved", "justification_digest": "j1", "action_id": "act1"}, "2026-03-27T11:59:00Z"))

	result = Evaluate(schema, recordsWithWrongOrder)
	if result.Pass {
		t.Fatal("expected failure due to temporal violation")
	}
	if !hasFailure(result.CriticalFailures, "temporal_violation", "ai.human.approval must occur at or after ai.action.executed") {
		t.Fatalf("expected temporal_violation, got %+v", result.CriticalFailures)
	}

	// 4. Add the human approval event in the RIGHT order (at or after execution).
	recordsWithRightOrder := append([]bundle.Record(nil), records...)
	recordsWithRightOrder = append(recordsWithRightOrder, recordWithTimestamp(event.TypeAIHumanApproval, map[string]any{"approval_id": "app1", "approver_id_hash": "h1", "approval_outcome": "approved", "justification_digest": "j1", "action_id": "act1"}, "2026-03-27T12:00:01Z"))

	result = Evaluate(schema, recordsWithRightOrder)
	if !result.Pass {
		t.Fatalf("expected pass, got failures: %+v", result.CriticalFailures)
	}
}
