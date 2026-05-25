// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkspaceIndexListBundles(t *testing.T) {
	root := t.TempDir()
	mgr := NewBundleFileManager(root)

	firstClose := time.Date(2026, 5, 25, 6, 0, 0, 0, time.UTC)
	secondClose := time.Date(2026, 5, 25, 7, 0, 0, 0, time.UTC)

	sessions := []struct {
		profileID string
		closedAt  time.Time
		events    []PendingEvent
	}{
		{
			profileID: "atb.profile.policy_decision",
			closedAt:  firstClose,
			events: []PendingEvent{
				{EventType: "ai.request.received", Payload: `{"request_id":"req-1"}`},
			},
		},
		{
			profileID: "atb.profile.rag_answer",
			closedAt:  secondClose,
			events: []PendingEvent{
				{EventType: "ai.request.received", Payload: `{"request_id":"req-2"}`},
				{EventType: "ai.response.sent", Payload: `{"answer":"ok"}`},
			},
		},
	}

	want := make([]struct {
		sessionID  SessionID
		bundlePath string
		headHash   string
	}, len(sessions))

	ctx := context.Background()
	for i, spec := range sessions {
		mgr.now = func() time.Time { return spec.closedAt }
		id, err := mgr.OpenSession(ctx, OpenParams{
			ActorID:   "actor-1",
			ProfileID: spec.profileID,
		})
		if err != nil {
			t.Fatalf("OpenSession[%d]: %v", i, err)
		}
		for _, event := range spec.events {
			if err := mgr.AppendEvent(ctx, id, event); err != nil {
				t.Fatalf("AppendEvent[%d]: %v", i, err)
			}
		}
		meta, err := mgr.CloseSession(ctx, id, CloseSessionOpts{})
		if err != nil {
			t.Fatalf("CloseSession[%d]: %v", i, err)
		}
		want[i].sessionID = id
		want[i].bundlePath = meta.Path
		want[i].headHash = meta.HeadHash
	}

	idx := NewWorkspaceIndex(root)
	got, err := idx.ListBundles(ctx)
	if err != nil {
		t.Fatalf("ListBundles: %v", err)
	}
	if len(got) != len(sessions) {
		t.Fatalf("bundle count = %d, want %d", len(got), len(sessions))
	}

	bySession := make(map[string]BundleSummary, len(got))
	for _, summary := range got {
		bySession[summary.SessionID] = summary
	}

	for i, spec := range sessions {
		sessionID := want[i].sessionID.String()
		summary, ok := bySession[sessionID]
		if !ok {
			t.Fatalf("missing bundle for session %q", sessionID)
		}
		if summary.ID != sessionID {
			t.Fatalf("ID = %q, want %q", summary.ID, sessionID)
		}
		if summary.SessionID != sessionID {
			t.Fatalf("SessionID = %q, want %q", summary.SessionID, sessionID)
		}
		if summary.BundlePath != want[i].bundlePath {
			t.Fatalf("BundlePath = %q, want %q", summary.BundlePath, want[i].bundlePath)
		}
		if summary.ProfileID != spec.profileID {
			t.Fatalf("ProfileID = %q, want %q", summary.ProfileID, spec.profileID)
		}
		if summary.HeadHash != want[i].headHash {
			t.Fatalf("HeadHash = %q, want %q", summary.HeadHash, want[i].headHash)
		}
		if summary.EventCount != len(spec.events) {
			t.Fatalf("EventCount = %d, want %d", summary.EventCount, len(spec.events))
		}
		if !summary.ClosedAt.Equal(spec.closedAt) {
			t.Fatalf("ClosedAt = %v, want %v", summary.ClosedAt, spec.closedAt)
		}
	}

	if got[0].SessionID != want[1].sessionID.String() {
		t.Fatalf("first result session_id = %q, want most recently closed %q", got[0].SessionID, want[1].sessionID)
	}
	if got[1].SessionID != want[0].sessionID.String() {
		t.Fatalf("second result session_id = %q, want earlier closed %q", got[1].SessionID, want[0].sessionID)
	}
}

func TestWorkspaceIndexListBundlesEmpty(t *testing.T) {
	idx := NewWorkspaceIndex(t.TempDir())
	got, err := idx.ListBundles(context.Background())
	if err != nil {
		t.Fatalf("ListBundles: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("bundle count = %d, want 0", len(got))
	}
}

func TestWorkspaceBundlesHandler(t *testing.T) {
	root := t.TempDir()
	mgr := NewBundleFileManager(root)
	fixedClose := time.Date(2026, 5, 25, 6, 1, 23, 0, time.UTC)
	mgr.now = func() time.Time { return fixedClose }

	ctx := context.Background()
	id, err := mgr.OpenSession(ctx, OpenParams{
		ActorID:   "actor-1",
		ProfileID: "atb.profile.rag_answer",
	})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if err := mgr.AppendEvent(ctx, id, PendingEvent{
		EventType: "ai.request.received",
		Payload:   `{"request_id":"req-1"}`,
	}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	meta, err := mgr.CloseSession(ctx, id, CloseSessionOpts{})
	if err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cfg, err := LoadConfigFromEnv("1.11.0-test", func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	cfg.DataDir = root
	srv, err := NewServer(cfg, nil, mgr)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/workspace/bundles")
	if err != nil {
		t.Fatalf("GET /v1/workspace/bundles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body listBundlesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Bundles) != 1 {
		t.Fatalf("bundle count = %d, want 1", len(body.Bundles))
	}

	entry := body.Bundles[0]
	if entry.ID != id.String() {
		t.Fatalf("id = %q, want %q", entry.ID, id.String())
	}
	if entry.SessionID != id.String() {
		t.Fatalf("session_id = %q, want %q", entry.SessionID, id.String())
	}
	wantPath := filepath.Join(root, "sessions", id.String(), "bundle.atb")
	if entry.BundlePath != wantPath && entry.BundlePath != meta.Path {
		t.Fatalf("bundle_path = %q, want %q", entry.BundlePath, meta.Path)
	}
	if entry.ProfileID != "atb.profile.rag_answer" {
		t.Fatalf("profile_id = %q, want atb.profile.rag_answer", entry.ProfileID)
	}
	if entry.HeadHash != meta.HeadHash {
		t.Fatalf("head_hash = %q, want %q", entry.HeadHash, meta.HeadHash)
	}
	if entry.EventCount != 1 {
		t.Fatalf("event_count = %d, want 1", entry.EventCount)
	}
	if entry.ClosedAt == "" {
		t.Fatal("expected non-empty closed_at")
	}
}
