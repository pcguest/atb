package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProfile_LoadsYAMLProfileAndEvaluatesMatchingBundle(t *testing.T) {
	profilePath := writeProfileFixture(t, `
id: "org.example.my_profile"
display_name: "My Custom Profile"
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
