// SPDX-License-Identifier: MIT
package incident_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/incident"
)

func writeBundle(t *testing.T) string {
	t.Helper()
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	add := func(typ string, data map[string]any) {
		if err := b.AppendWithOptions(typ, data, &bundle.AppendOptions{Timestamp: "2026-05-28T09:00:00Z"}); err != nil {
			t.Fatalf("append %s: %v", typ, err)
		}
	}
	// Session A: a privileged tool call with no approval, then a failure.
	add(event.TypeLLMRequest, map[string]any{"session_id": "sess-A", "host": "api.anthropic.com", "method": "POST", "path": "/v1/messages"})
	add(event.TypeToolCall, map[string]any{"session_id": "sess-A", "tool_name": "wipe_db"})
	add(event.TypeAIActionError, map[string]any{"session_id": "sess-A", "action_id": "toolu_1", "error_class": "failed"})
	add(event.TypeSessionClose, map[string]any{"session_id": "sess-A", "actor_id": "agent-x"})
	// Session B: unrelated, should be excluded.
	add(event.TypeLLMRequest, map[string]any{"session_id": "sess-B", "host": "api.openai.com", "method": "POST", "path": "/v1/chat/completions"})

	path := filepath.Join(t.TempDir(), "capture.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	return path
}

func TestBuildScopesSessionAndReportsIntegrity(t *testing.T) {
	path := writeBundle(t)

	rep, err := incident.Build(context.Background(), path, "sess-A")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !rep.Found {
		t.Fatal("want Found for sess-A")
	}
	if !rep.IntegrityValid {
		t.Error("want integrity valid for an untampered bundle")
	}
	if rep.ChainHeadHash == "" {
		t.Error("want a chain head hash")
	}
	// Only session A's four events, in sequence order; session B excluded.
	if len(rep.Events) != 4 {
		t.Fatalf("want 4 events for sess-A, got %d", len(rep.Events))
	}
	for i := 1; i < len(rep.Events); i++ {
		if rep.Events[i].Seq < rep.Events[i-1].Seq {
			t.Error("events not in sequence order")
		}
	}
	for _, e := range rep.Events {
		if e.Hash == "" {
			t.Errorf("event seq %d missing record hash", e.Seq)
		}
	}
	// The session summary should carry the tool_without_approval anomaly.
	if rep.Session == nil {
		t.Fatal("want a session summary")
	}
	found := false
	for _, f := range rep.Session.AnomalyFlags {
		if f == "tool_without_approval" {
			found = true
		}
	}
	if !found {
		t.Errorf("want tool_without_approval anomaly, got %v", rep.Session.AnomalyFlags)
	}

	md := rep.Markdown()
	if !strings.Contains(md, "tool=wipe_db") || !strings.Contains(md, "error_class=failed") {
		t.Errorf("markdown missing event summaries:\n%s", md)
	}
}

func TestBuildSessionNotFound(t *testing.T) {
	path := writeBundle(t)
	rep, err := incident.Build(context.Background(), path, "sess-missing")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if rep.Found {
		t.Error("want Found=false for an unknown session")
	}
	if len(rep.Events) != 0 {
		t.Errorf("want no events, got %d", len(rep.Events))
	}
}

func TestReportJSONRoundTrips(t *testing.T) {
	path := writeBundle(t)
	rep, err := incident.Build(context.Background(), path, "sess-A")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	raw, err := rep.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var back incident.Report
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.SessionID != "sess-A" || len(back.Events) != 4 {
		t.Errorf("json round-trip mismatch: %+v", back)
	}
}
