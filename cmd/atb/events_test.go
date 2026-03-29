package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

// TestRunEvents_DefaultOutput verifies that the default output contains
// known event types.
func TestRunEvents_DefaultOutput(t *testing.T) {
	out := captureEventsOutput(t, []string{})
	if !strings.Contains(out, event.TypeBundleManifest) {
		t.Error("expected atb.bundle.manifest in default output")
	}
	if !strings.Contains(out, event.TypeAIActionPrecommit) {
		t.Error("expected ai.action.precommit in default output")
	}
}

// TestRunEvents_JSONOutput verifies that --json emits a valid JSON array
// matching the full registry.
func TestRunEvents_JSONOutput(t *testing.T) {
	out := captureEventsOutput(t, []string{"--json"})

	var items []map[string]any
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("--json output is not valid JSON: %v", err)
	}
	if len(items) != len(event.Registry) {
		t.Errorf("expected %d items, got %d", len(event.Registry), len(items))
	}
}

// TestRunEvents_ProfileFilter verifies that --profile filters output.
func TestRunEvents_ProfileFilter(t *testing.T) {
	out := captureEventsOutput(t, []string{"--profile", "atb.profile.rag_answer"})
	if !strings.Contains(out, event.TypeAIModelInvoked) {
		t.Error("expected ai.model.invoked in rag_answer filter output")
	}
	if strings.Contains(out, event.TypeAIActionPrecommit) {
		t.Error("ai.action.precommit should not appear when filtering by rag_answer profile")
	}
}

// captureEventsOutput runs the events command and returns stdout as a string.
func captureEventsOutput(t *testing.T, args []string) string {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvents(args, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvents returned exit code %d, stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}
	return stdout.String()
}
