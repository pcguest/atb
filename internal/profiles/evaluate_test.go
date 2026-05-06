// SPDX-License-Identifier: MIT
package profiles

import (
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func TestEvaluate_MissingRequired(t *testing.T) {
	schema := testSchema()

	result := Evaluate(schema, nil)
	if result.Pass {
		t.Fatal("expected failure")
	}
	if !hasFailure(result.CriticalFailures, "missing_event", "ai.request.received required") {
		t.Fatalf("expected missing required failure, got %+v", result.CriticalFailures)
	}
	gotID := failureID(result.CriticalFailures, "missing_event", "ai.request.received required")
	if gotID != "required:ai.request.received" {
		t.Fatalf("missing required ID = %q, want %q", gotID, "required:ai.request.received")
	}
}

func TestEvaluate_MissingRequiredFieldID(t *testing.T) {
	schema := testSchema()
	records := []bundle.Record{
		record("ai.request.received", map[string]any{}),
	}

	result := Evaluate(schema, records)
	if result.Pass {
		t.Fatal("expected failure")
	}
	gotID := failureID(result.CriticalFailures, "missing_field", "ai.request.received required")
	if gotID != "required:ai.request.received:field:request_id" {
		t.Fatalf("missing required field ID = %q, want %q", gotID, "required:ai.request.received:field:request_id")
	}
}

func TestEvaluate_MissingOptional(t *testing.T) {
	schema := testSchema()
	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if !contains(result.RequiredWarnings, "ai.response.sent recommended") {
		t.Fatalf("expected optional warning, got %+v", result.RequiredWarnings)
	}
}

func TestEvaluate_OptionalCriticalProducesFailure(t *testing.T) {
	schema := testSchema()
	schema.Optional[0].Severity = "critical"

	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if !hasFailure(result.CriticalFailures, "missing_event", "ai.response.sent recommended") {
		t.Fatalf("expected critical failure, got %+v", result.CriticalFailures)
	}
	if contains(result.RequiredWarnings, "ai.response.sent recommended") {
		t.Fatalf("expected no warning duplication, got %+v", result.RequiredWarnings)
	}
}

func TestEvaluate_RequiredWarningProducesWarning(t *testing.T) {
	schema := testSchema()
	schema.Required[0].Severity = "warning"

	result := Evaluate(schema, nil)
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", result.CriticalFailures)
	}
	if !contains(result.RequiredWarnings, "ai.request.received required") {
		t.Fatalf("expected warning, got %+v", result.RequiredWarnings)
	}
}

func TestEvaluate_RelationViolation(t *testing.T) {
	schema := testSchema()
	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
		record("ai.response.sent", map[string]any{"request_id": "req-2"}),
	}

	result := Evaluate(schema, records)
	if !hasFailure(result.CriticalFailures, "relation_violation", "request must match response") {
		t.Fatalf("expected relation failure, got %+v", result.CriticalFailures)
	}
	gotID := failureID(result.CriticalFailures, "relation_violation", "request must match response")
	if gotID != "relation:request_to_response" {
		t.Fatalf("relation failure ID = %q, want %q", gotID, "relation:request_to_response")
	}
}

func TestEvaluate_RelationMissingTargetSkipped(t *testing.T) {
	schema := testSchema()
	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if hasFailure(result.CriticalFailures, "relation_violation", "request must match response") {
		t.Fatalf("did not expect relation failure when target event type is absent, got %+v", result.CriticalFailures)
	}
}

func TestEvaluate_RelationPredicatePolicyDecision(t *testing.T) {
	const predicateID = "relation:execution_after_authorization:predicate:decision:allow"

	tests := []struct {
		name      string
		schemaID  string
		records   []bundle.Record
		wantPass  bool
		wantError bool
	}{
		{
			name:      "privileged tool action deny fails",
			schemaID:  "atb.profile.privileged_tool_action",
			records:   privilegedToolActionRecords("deny"),
			wantError: true,
		},
		{
			name:      "data export deny fails",
			schemaID:  "atb.profile.data_export",
			records:   dataExportRecords("deny"),
			wantError: true,
		},
		{
			name:     "privileged tool action allow passes",
			schemaID: "atb.profile.privileged_tool_action",
			records:  privilegedToolActionRecords("allow"),
			wantPass: true,
		},
		{
			name:     "data export allow passes",
			schemaID: "atb.profile.data_export",
			records:  dataExportRecords("allow"),
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Evaluate(MustLoadSchema(tt.schemaID), tt.records)
			if tt.wantPass && !result.Pass {
				t.Fatalf("expected pass, got failures %+v", result.CriticalFailures)
			}
			if tt.wantError && !hasFailureID(result.CriticalFailures, predicateID) {
				t.Fatalf("expected predicate failure ID %q, got %+v", predicateID, result.CriticalFailures)
			}
		})
	}
}

func TestEvaluate_Clean(t *testing.T) {
	schema := testSchema()
	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
		record("ai.response.sent", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if !result.Pass {
		t.Fatalf("expected pass, got %+v", result.CriticalFailures)
	}
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no failures, got %+v", result.CriticalFailures)
	}
	if len(result.RequiredWarnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", result.RequiredWarnings)
	}
}

func TestEvaluate_RequiredWhenSatisfied(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Severity: "warning",
				RequiredWhen: []requiredWhenRule{
					{WhenType: "ai.request.received", AtOrAfter: true},
				},
			},
		},
	}

	records := []bundle.Record{
		recordWithTimestamp("ai.request.received", map[string]any{"request_id": "req-1"}, "2026-03-27T12:00:00Z"),
		recordWithTimestamp("ai.response.sent", map[string]any{"request_id": "req-1"}, "2026-03-27T12:01:00Z"),
	}

	result := Evaluate(schema, records)
	if !result.Pass {
		t.Fatalf("expected pass, got failures: %+v", result.CriticalFailures)
	}
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", result.CriticalFailures)
	}
}

func TestEvaluate_RequiredWhenMissingTarget(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Severity: "warning",
				RequiredWhen: []requiredWhenRule{
					{WhenType: "ai.request.received"},
				},
			},
		},
	}

	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if result.Pass {
		t.Fatal("expected failure")
	}
	if !hasFailure(result.CriticalFailures, "missing_event", "ai.response.sent") {
		t.Fatalf("expected missing_event for ai.response.sent, got %+v", result.CriticalFailures)
	}
	gotID := failureID(result.CriticalFailures, "missing_event", "ai.response.sent")
	if !strings.HasPrefix(gotID, "required_when:") {
		t.Fatalf("required_when failure ID = %q, want prefix %q", gotID, "required_when:")
	}
}

func TestEvaluate_RequiredWhenConditionNotMet(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Severity: "warning",
				RequiredWhen: []requiredWhenRule{
					{WhenType: "ai.request.received"},
				},
			},
		},
	}

	records := []bundle.Record{
		record("ai.response.sent", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", result.CriticalFailures)
	}
}

func TestEvaluate_RequiredWhenAtOrAfterViolation(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Severity: "warning",
				RequiredWhen: []requiredWhenRule{
					{WhenType: "ai.request.received", AtOrAfter: true},
				},
			},
		},
	}

	records := []bundle.Record{
		recordWithTimestamp("ai.response.sent", map[string]any{"request_id": "req-1"}, "2026-03-27T11:59:00Z"),
		recordWithTimestamp("ai.request.received", map[string]any{"request_id": "req-1"}, "2026-03-27T12:00:00Z"),
	}

	result := Evaluate(schema, records)
	if result.Pass {
		t.Fatal("expected failure")
	}
	if !hasFailure(result.CriticalFailures, "temporal_violation", "ai.response.sent") {
		t.Fatalf("expected temporal_violation for ai.response.sent, got %+v", result.CriticalFailures)
	}
}

func TestEvaluate_RequiredWhenMissingConditionTimestampWarns(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Severity: "warning",
				RequiredWhen: []requiredWhenRule{
					{WhenType: "ai.request.received", AtOrAfter: true},
				},
			},
		},
	}

	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
		recordWithTimestamp("ai.response.sent", map[string]any{"request_id": "req-1"}, "2026-03-27T12:01:00Z"),
	}

	result := Evaluate(schema, records)
	if !result.Pass {
		t.Fatalf("expected pass with warning, got failures %+v", result.CriticalFailures)
	}
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", result.CriticalFailures)
	}
	if !containsSubstring(result.RequiredWarnings, "unable to enforce ordering between ai.response.sent and ai.request.received because ai.request.received timestamp is missing") {
		t.Fatalf("expected timestamp warning, got %+v", result.RequiredWarnings)
	}
}

func TestEvaluate_RequiredWhenAtOrAfterBadTS(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Severity: "warning",
				RequiredWhen: []requiredWhenRule{
					{WhenType: "ai.request.received", AtOrAfter: true},
				},
			},
		},
	}

	records := []bundle.Record{
		recordWithTimestamp("ai.request.received", map[string]any{"request_id": "req-1"}, "2026-03-27T12:00:00Z"),
		record("ai.response.sent", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if !result.Pass {
		t.Fatalf("expected pass with warning, got failures %+v", result.CriticalFailures)
	}
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", result.CriticalFailures)
	}
	if !containsSubstring(result.RequiredWarnings, "unable to enforce ordering between ai.response.sent and ai.request.received because ai.response.sent timestamps are missing or invalid") {
		t.Fatalf("expected target timestamp warning, got %+v", result.RequiredWarnings)
	}
}

func TestEvaluate_NoRequiredWhenRules(t *testing.T) {
	schema := testSchema()
	records := []bundle.Record{
		record("ai.request.received", map[string]any{"request_id": "req-1"}),
	}

	result := Evaluate(schema, records)
	if !result.Pass {
		t.Fatalf("expected pass, got failures %+v", result.CriticalFailures)
	}
	if len(result.CriticalFailures) != 0 {
		t.Fatalf("expected no critical failures, got %+v", result.CriticalFailures)
	}
	if len(result.RequiredWarnings) != 1 || result.RequiredWarnings[0] != "ai.response.sent recommended" {
		t.Fatalf("unexpected required warnings: %+v", result.RequiredWarnings)
	}
	if len(result.InformationalNotes) != 0 {
		t.Fatalf("expected no informational notes, got %+v", result.InformationalNotes)
	}
}

func testSchema() ProfileSchema {
	return ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Required: []EventRule{
			{
				Type:     "ai.request.received",
				Fields:   []string{"request_id"},
				Message:  "ai.request.received required",
				Severity: "critical",
			},
		},
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Fields:   []string{"request_id"},
				Message:  "ai.response.sent recommended",
				Severity: "warning",
			},
		},
		Relations: []RelationRule{
			{
				Name:    "request_to_response",
				From:    "ai.response.sent",
				To:      "ai.request.received",
				Field:   "request_id",
				Message: "request must match response",
			},
		},
	}
}

func privilegedToolActionRecords(decision string) []bundle.Record {
	return []bundle.Record{
		record("atb.bundle.manifest", map[string]any{}),
		record("ai.request.received", map[string]any{
			"request_id":    "req-1",
			"actor_id_hash": "actor-1",
			"purpose_tag":   "privileged_tool_action",
		}),
		record("ai.action.precommit", map[string]any{
			"action_id":                "act-1",
			"action_type":              "write",
			"action_parameters_digest": "sha256:params",
			"target_resource_id":       "resource-1",
			"intended_effect":          "mutate resource",
		}),
		policyDecisionRecord(decision),
		record("ai.action.executed", map[string]any{
			"action_id":           "act-1",
			"execution_outcome":   "success",
			"tool_receipt_digest": "sha256:tool",
		}),
		record("ai.action.committed", map[string]any{
			"action_id":           "act-1",
			"commit_outcome":      "success",
			"sink_receipt_digest": "sha256:sink",
		}),
		humanApprovalRecord(),
	}
}

func dataExportRecords(decision string) []bundle.Record {
	return []bundle.Record{
		record("atb.bundle.manifest", map[string]any{}),
		record("ai.request.received", map[string]any{
			"request_id":    "req-1",
			"actor_id_hash": "actor-1",
			"purpose_tag":   "data_export",
		}),
		policyDecisionRecord(decision),
		record("data.export.precommit", map[string]any{
			"action_id":                "act-1",
			"action_type":              "export",
			"action_parameters_digest": "sha256:params",
			"target_resource_id":       "dataset-1",
			"intended_effect":          "export dataset",
		}),
		record("data.export.executed", map[string]any{
			"action_id":           "act-1",
			"execution_outcome":   "success",
			"tool_receipt_digest": "sha256:tool",
		}),
		humanApprovalRecord(),
	}
}

func policyDecisionRecord(decision string) bundle.Record {
	return record("ai.policy.decision", map[string]any{
		"policy_id":             "policy-1",
		"policy_version":        "v1",
		"decision":              decision,
		"decision_reason_codes": []string{"matched"},
		"subject_id_hash":       "actor-1",
		"action_id":             "act-1",
	})
}

func humanApprovalRecord() bundle.Record {
	return record("ai.human.approval", map[string]any{
		"approval_id":          "approval-1",
		"approver_id_hash":     "approver-1",
		"approval_outcome":     "approved",
		"justification_digest": "sha256:justification",
		"action_id":            "act-1",
	})
}

func record(eventType string, data map[string]any) bundle.Record {
	return bundle.Record{
		Event: hash.Event{
			Type: eventType,
			Data: data,
		},
	}
}

func recordWithTimestamp(eventType string, data map[string]any, timestamp string) bundle.Record {
	return bundle.Record{
		Event: hash.Event{
			Type:      eventType,
			Data:      data,
			Timestamp: timestamp,
		},
	}
}

func hasFailure(failures []CriticalFailure, kind string, containsText string) bool {
	for _, failure := range failures {
		if failure.Kind == kind && strings.Contains(failure.Detail, containsText) {
			return true
		}
	}
	return false
}

func failureID(failures []CriticalFailure, kind string, containsText string) string {
	for _, failure := range failures {
		if failure.Kind == kind && strings.Contains(failure.Detail, containsText) {
			return failure.ID
		}
	}
	return ""
}

func hasFailureID(failures []CriticalFailure, id string) bool {
	for _, failure := range failures {
		if failure.ID == id {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
