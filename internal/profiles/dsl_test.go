package profiles

import (
	"strings"
	"testing"
)

func TestParseDSLProfile_Valid_WithCASWeights(t *testing.T) {
	yaml := `
id: "org.example.custom"
version: 2
description: "Custom support workflow"
required_events:
  - "ai.request.received"
  - "ai.action.precommit"
warning_events:
  - "ai.action.committed"
cas_weights:
  EC: 0.30
  FC: 0.20
  RC: 0.10
  TC: 0.10
  SC: 0.10
  XC: 0.10
  AC: 0.05
  GC: 0.05
`
	schema, err := ParseDSLProfile([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseDSLProfile: %v", err)
	}

	if schema.ID != "org.example.custom" {
		t.Errorf("ID = %q, want %q", schema.ID, "org.example.custom")
	}
	if schema.Version != 2 {
		t.Errorf("Version = %d, want 2", schema.Version)
	}
	if schema.WorkflowClass != "Custom support workflow" {
		t.Errorf("WorkflowClass = %q, want %q", schema.WorkflowClass, "Custom support workflow")
	}
	if !schema.SupportsCAS {
		t.Errorf("SupportsCAS = false, want true")
	}
	if len(schema.Required) != 2 {
		t.Fatalf("len(Required) = %d, want 2", len(schema.Required))
	}
	if schema.Required[0].Type != "ai.request.received" {
		t.Errorf("Required[0].Type = %q, want %q", schema.Required[0].Type, "ai.request.received")
	}
	if schema.Required[0].Severity != "critical" {
		t.Errorf("Required[0].Severity = %q, want %q", schema.Required[0].Severity, "critical")
	}
	if schema.Required[1].Type != "ai.action.precommit" {
		t.Errorf("Required[1].Type = %q, want %q", schema.Required[1].Type, "ai.action.precommit")
	}
	if len(schema.Optional) != 1 {
		t.Fatalf("len(Optional) = %d, want 1", len(schema.Optional))
	}
	if schema.Optional[0].Type != "ai.action.committed" {
		t.Errorf("Optional[0].Type = %q, want %q", schema.Optional[0].Type, "ai.action.committed")
	}
	if schema.Optional[0].Severity != "warning" {
		t.Errorf("Optional[0].Severity = %q, want %q", schema.Optional[0].Severity, "warning")
	}
	if len(schema.Weights) != 8 {
		t.Errorf("len(Weights) = %d, want 8", len(schema.Weights))
	}
	if schema.Weights["EC"] != 0.30 {
		t.Errorf("Weights[EC] = %f, want 0.30", schema.Weights["EC"])
	}
}

func TestParseDSLProfile_Valid_NoCASWeights(t *testing.T) {
	yaml := `
id: "org.example.minimal"
required_events:
  - "ai.request.received"
`
	schema, err := ParseDSLProfile([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseDSLProfile: %v", err)
	}
	if schema.ID != "org.example.minimal" {
		t.Errorf("ID = %q, want %q", schema.ID, "org.example.minimal")
	}
	if schema.SupportsCAS {
		t.Errorf("SupportsCAS = true, want false (no cas_weights)")
	}
	// All 8 weight keys must still be present (zeroed).
	if len(schema.Weights) != 8 {
		t.Errorf("len(Weights) = %d, want 8", len(schema.Weights))
	}
	for _, k := range []string{"EC", "FC", "RC", "TC", "SC", "XC", "AC", "GC"} {
		if v, ok := schema.Weights[k]; !ok || v != 0.0 {
			t.Errorf("Weights[%s] = %f ok=%v, want 0.0 true", k, v, ok)
		}
	}
}

func TestParseDSLProfile_Valid_VersionDefaults(t *testing.T) {
	yaml := `
id: "org.example.noversion"
required_events:
  - "ai.request.received"
`
	schema, err := ParseDSLProfile([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseDSLProfile: %v", err)
	}
	if schema.Version != 1 {
		t.Errorf("Version = %d, want 1 (default)", schema.Version)
	}
}

func TestParseDSLProfile_Valid_WorkflowClassFromID(t *testing.T) {
	yaml := `
id: "acme.corp.profiles.my_workflow"
required_events:
  - "ai.request.received"
`
	schema, err := ParseDSLProfile([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseDSLProfile: %v", err)
	}
	// workflowClassFromID should return the last dot-segment.
	if schema.WorkflowClass != "my_workflow" {
		t.Errorf("WorkflowClass = %q, want %q", schema.WorkflowClass, "my_workflow")
	}
}

func TestParseDSLProfile_Valid_PartialCASWeights(t *testing.T) {
	// Partial weights: only some keys specified, unspecified → 0.
	yaml := `
id: "org.example.partial"
required_events:
  - "ai.request.received"
cas_weights:
  EC: 0.60
  FC: 0.40
`
	schema, err := ParseDSLProfile([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseDSLProfile: %v", err)
	}
	if !schema.SupportsCAS {
		t.Errorf("SupportsCAS = false, want true")
	}
	if schema.Weights["EC"] != 0.60 {
		t.Errorf("Weights[EC] = %f, want 0.60", schema.Weights["EC"])
	}
	if schema.Weights["FC"] != 0.40 {
		t.Errorf("Weights[FC] = %f, want 0.40", schema.Weights["FC"])
	}
	if schema.Weights["RC"] != 0.0 {
		t.Errorf("Weights[RC] = %f, want 0.0", schema.Weights["RC"])
	}
}

func TestParseDSLProfile_Error_MissingID(t *testing.T) {
	yaml := `
required_events:
  - "ai.request.received"
`
	_, err := ParseDSLProfile([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing id")
	}
	if !strings.Contains(err.Error(), "id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDSLProfile_Error_CollidesWithBuiltIn(t *testing.T) {
	yaml := `
id: "atb.profile.privileged_tool_action"
required_events:
  - "ai.request.received"
`
	_, err := ParseDSLProfile([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for built-in ID collision")
	}
	if !strings.Contains(err.Error(), "collides with a built-in profile") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDSLProfile_Error_CASWeightsNotSumToOne(t *testing.T) {
	yaml := `
id: "org.example.bad_weights"
required_events:
  - "ai.request.received"
cas_weights:
  EC: 0.50
  FC: 0.30
`
	_, err := ParseDSLProfile([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for weights not summing to 1.0")
	}
	if !strings.Contains(err.Error(), "cas_weights must sum to 1.0") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDSLProfile_Error_UnknownCASKey(t *testing.T) {
	yaml := `
id: "org.example.unknown_key"
required_events:
  - "ai.request.received"
cas_weights:
  ZZ: 0.50
  EC: 0.50
`
	_, err := ParseDSLProfile([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for unknown CAS key")
	}
	if !strings.Contains(err.Error(), "unknown CAS weight key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDSLProfile_Error_EmptyEventInRequiredEvents(t *testing.T) {
	yaml := `
id: "org.example.empty_event"
required_events:
  - "ai.request.received"
  - ""
`
	_, err := ParseDSLProfile([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for empty event type")
	}
	if !strings.Contains(err.Error(), "empty event type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDSLProfile_Error_DuplicateRequiredEvent(t *testing.T) {
	yaml := `
id: "org.example.dup_event"
required_events:
  - "ai.request.received"
  - "ai.request.received"
`
	_, err := ParseDSLProfile([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for duplicate event type")
	}
	if !strings.Contains(err.Error(), "duplicate event type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDSLProfile_Error_InvalidYAML(t *testing.T) {
	_, err := ParseDSLProfile([]byte(":\tinvalid: yaml: [\n"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestWorkflowClassFromID(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"demo", "demo"},
		{"atb.profile.custom.demo", "demo"},
		{"a.b.c", "c"},
	}
	for _, tc := range cases {
		got := workflowClassFromID(tc.id)
		if got != tc.want {
			t.Errorf("workflowClassFromID(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
