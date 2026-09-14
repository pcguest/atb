// SPDX-License-Identifier: MIT
package incident_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/incident"
	"github.com/pcguest/atb/internal/sessionindex"
)

func findingByFlag(findings []incident.Finding, flag string) *incident.Finding {
	for i := range findings {
		if findings[i].Flag == flag {
			return &findings[i]
		}
	}
	return nil
}

func TestBuildBundleFindings_ScopesToLoadedBundle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bundleA := filepath.Join(dir, "bundle-a.atb")
	bundleB := filepath.Join(dir, "bundle-b.atb")

	bA, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	if err := bA.Append(event.TypeToolCall, map[string]any{
		"session_id": "sess-a",
		"tool_name":  "read_file",
	}); err != nil {
		t.Fatalf("append A: %v", err)
	}
	if err := bA.Save(bundleA); err != nil {
		t.Fatalf("save A: %v", err)
	}

	bB, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	if err := bB.Append(event.TypeToolCall, map[string]any{
		"session_id": "sess-b",
		"tool_name":  "write_file",
	}); err != nil {
		t.Fatalf("append B: %v", err)
	}
	if err := bB.Save(bundleB); err != nil {
		t.Fatalf("save B: %v", err)
	}

	entries, err := sessionindex.BuildIndex(context.Background(), []string{bundleA, bundleB})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries count = %d, want 2", len(entries))
	}

	// Correlating against bundleA must never produce findings for session-b in bundleB
	findingsA := incident.BuildBundleFindings(bA, bundleA, entries)
	for _, f := range findingsA {
		if f.SessionID != "sess-a" {
			t.Fatalf("findings for bundleA cited unexpected session %q", f.SessionID)
		}
	}
	twa := findingByFlag(findingsA, "tool_without_approval")
	if twa == nil {
		t.Fatal("expected tool_without_approval finding for bundleA")
	}
	if len(twa.EventSeqs) != 1 || twa.EventSeqs[0] != 1 {
		t.Fatalf("finding on bundleA has unexpected EventSeqs: %v, want [1]", twa.EventSeqs)
	}
}

func TestBuildBundleFindings_CorrelatesEventsWithoutSessionID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "no-session.atb")

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	// Append tool call without any session_id in payload
	if err := b.Append(event.TypeToolCall, map[string]any{
		"tool_name": "bash",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save: %v", err)
	}

	entries, err := sessionindex.BuildIndex(context.Background(), []string{bundlePath})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries count = %d, want 1 fallback session", len(entries))
	}
	if entries[0].SessionID != "no-session" {
		t.Fatalf("sessionID = %q, want 'no-session'", entries[0].SessionID)
	}

	findings := incident.BuildBundleFindings(b, bundlePath, entries)
	twa := findingByFlag(findings, "tool_without_approval")
	if twa == nil {
		t.Fatal("expected tool_without_approval finding for unapproved tool call")
	}
	if twa.SessionID != "no-session" {
		t.Fatalf("finding SessionID = %q, want 'no-session'", twa.SessionID)
	}
	// Crucial check: event sequence must be non-empty and point to the unapproved tool call
	if len(twa.EventSeqs) != 1 || twa.EventSeqs[0] != 1 {
		t.Fatalf("finding EventSeqs = %v, want [1]", twa.EventSeqs)
	}
}
