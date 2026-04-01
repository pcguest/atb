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

func testSchema() ProfileSchema {
	return ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights: map[string]float64{
			"EC": 0.20,
			"FC": 0.15,
			"RC": 0.20,
			"TC": 0.05,
			"SC": 0.10,
			"XC": 0.10,
			"AC": 0.10,
			"GC": 0.10,
		},
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
