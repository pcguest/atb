package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProfile_LoadsSchemaBackedYAMLProfileAndEvaluatesMatchingBundle(t *testing.T) {
	profilePath := writeProfileFixture(t, `
id: "org.example.my_profile"
version: 1
workflow_class: "custom_profile"
blind_spots:
  - "Custom profiles inherit the same blind-spot declaration surface as built-ins."
weights:
  EC: 0.20
  FC: 0.15
  RC: 0.20
  TC: 0.05
  SC: 0.10
  XC: 0.10
  AC: 0.10
  GC: 0.10
required:
  - type: "ai.action.precommit"
    fields:
      - "action_id"
    message: "Pre-commit record required"
    severity: "critical"
  - type: "ai.action.executed"
    fields:
      - "action_id"
    message: "Execution record required"
    severity: "critical"
optional:
  - type: "atb.bundle.anchor"
    fields: []
    message: "Anchor required"
    severity: "warning"
relations:
  - name: "precommit_to_execution"
    from: "ai.action.executed"
    to: "ai.action.precommit"
    field: "action_id"
    message: "Precommit must precede execution"
`)

	profile, err := ResolveProfile(profilePath)
	if err != nil {
		t.Fatalf("ResolveProfile(%q): %v", profilePath, err)
	}
	if profile == nil {
		t.Fatalf("expected loaded profile")
	}
	if profile.ID() != "org.example.my_profile" {
		t.Fatalf("unexpected profile ID: got %q want %q", profile.ID(), "org.example.my_profile")
	}

	report := VerifyWithProfile(newPrivilegedToolActionBundle(t), "bundle.atb", profile)
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one evaluated profile, got %d", len(report.Profiles))
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected custom profile pass, got failures %+v", report.Profiles[0].CriticalFailures)
	}
}

func TestResolveProfile_LoadsLegacyYAMLProfileAndEvaluatesMatchingBundle(t *testing.T) {
	profilePath := writeProfileFixture(t, `
id: "org.example.legacy_profile"
display_name: "My Legacy Profile"
description: "Custom obligations for our deployment"
detect:
  event_types:
    - "ai.action.precommit"
obligations:
  critical:
    - event_type: "ai.action.precommit"
      message: "Pre-commit record required"
    - event_type: "ai.action.executed"
      message: "Execution record required"
  required:
    - event_type: "atb.bundle.anchor"
      message: "Anchor required"
relations:
  - from: "ai.action.precommit"
    to: "ai.action.executed"
    message: "Precommit must precede execution"
`)

	profile, err := ResolveProfile(profilePath)
	if err != nil {
		t.Fatalf("ResolveProfile(%q): %v", profilePath, err)
	}
	if profile == nil {
		t.Fatalf("expected loaded profile")
	}
	if profile.ID() != "org.example.legacy_profile" {
		t.Fatalf("unexpected profile ID: got %q want %q", profile.ID(), "org.example.legacy_profile")
	}

	report := VerifyWithProfile(newPrivilegedToolActionBundle(t), "bundle.atb", profile)
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one evaluated profile, got %d", len(report.Profiles))
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected legacy custom profile pass, got failures %+v", report.Profiles[0].CriticalFailures)
	}
}

func TestResolveProfile_MissingIDReturnsError(t *testing.T) {
	profilePath := writeProfileFixture(t, `
display_name: "My Custom Profile"
obligations:
  critical:
    - event_type: "ai.action.precommit"
      message: "Pre-commit record required"
`)

	_, err := ResolveProfile(profilePath)
	if err == nil {
		t.Fatalf("expected error for missing id")
	}
	if !strings.Contains(err.Error(), `missing required field "id"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveProfile_FileNotFoundReturnsError(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "missing-profile.yaml")

	_, err := ResolveProfile(profilePath)
	if err == nil {
		t.Fatalf("expected file not found error")
	}
	if !strings.Contains(err.Error(), "load profile") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveProfile_BuiltInIDStillResolvesFromRegistry(t *testing.T) {
	profile, err := ResolveProfile(profileIDPrivilegedToolAction)
	if err != nil {
		t.Fatalf("ResolveProfile(%q): %v", profileIDPrivilegedToolAction, err)
	}
	if profile == nil {
		t.Fatalf("expected built-in profile")
	}
	if profile.ID() != profileIDPrivilegedToolAction {
		t.Fatalf("unexpected profile ID: got %q want %q", profile.ID(), profileIDPrivilegedToolAction)
	}
	if _, ok := profile.(*PrivilegedToolActionProfile); !ok {
		t.Fatalf("expected privileged tool action profile, got %T", profile)
	}
}

func writeProfileFixture(t testing.TB, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0600); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}
	return path
}
