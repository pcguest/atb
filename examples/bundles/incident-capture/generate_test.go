// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/pcguest/atb/internal/sessionindex"
)

// TestIncidentBundleIntegrityAndOversight pins the forensic incident demo:
//   - the capture-shaped bundle verifies (hash chain intact), and
//   - the session index raises the tool_without_approval anomaly, because the
//     model invoked a privileged tool (atb.tool.call) with no preceding
//     ai/atb.human.approval — the core oversight signal for agent incidents.
//
// It also confirms the new capture/accountability event types are present.
// Runs against the in-process builder so it is reproducible in CI without the
// gitignored .atb artefact.
func TestIncidentBundleIntegrityAndOversight(t *testing.T) {
	b, err := buildIncidentBundle()
	if err != nil {
		t.Fatalf("build incident bundle: %v", err)
	}

	if err := b.Verify(); err != nil {
		t.Fatalf("incident bundle failed integrity verification: %v", err)
	}

	// The bundle must contain the capture + accountability events.
	want := map[string]bool{
		"atb.llm.request":  false,
		"atb.llm.response": false,
		"atb.tool.call":    false,
		"ai.action.error":  false,
	}
	for _, rec := range b.Records {
		if _, ok := want[rec.Event.Type]; ok {
			want[rec.Event.Type] = true
		}
	}
	for typ, seen := range want {
		if !seen {
			t.Errorf("incident bundle missing expected event type %q", typ)
		}
	}

	// Persist and run the session index over it.
	path := filepath.Join(t.TempDir(), "incident-capture.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	entries, err := sessionindex.BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 session, got %d", len(entries))
	}
	flags := entries[0].AnomalyFlags
	if !slices.Contains(flags, "tool_without_approval") {
		t.Errorf("want tool_without_approval anomaly, got %v", flags)
	}
	if slices.Contains(flags, "session_not_closed") {
		t.Errorf("session should be closed; flags = %v", flags)
	}
}
