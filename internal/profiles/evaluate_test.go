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
