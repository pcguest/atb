// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryBundleManagerLifecycle(t *testing.T) {
	fixedNow := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	mgr := NewMemoryBundleManager(filepath.Join(t.TempDir(), "agent"))
	mgr.now = func() time.Time { return fixedNow }

	ctx := context.Background()
	id, err := mgr.OpenSession(ctx, OpenParams{
		ActorID:    "actor-1",
		PurposeTag: "support-triage",
		ProfileID:  "atb.profile.policy_decision",
	})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session id")
	}
	if mgr.ActiveSessionCount() != 1 {
		t.Fatalf("ActiveSessionCount = %d, want 1", mgr.ActiveSessionCount())
	}

	events := []PendingEvent{
		{EventType: "ai.request.received", Payload: `{"request_id":"req-1"}`},
		{EventType: "ai.policy.decision", Payload: `{"decision":"allow"}`},
	}
	for _, event := range events {
		if err := mgr.AppendEvent(ctx, id, event); err != nil {
			t.Fatalf("AppendEvent(%s): %v", event.EventType, err)
		}
	}

	meta, err := mgr.CloseSession(ctx, id, CloseSessionOpts{SnapshotName: "review_boundary"})
	if err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	if meta.SessionID != id {
		t.Fatalf("SessionID = %q, want %q", meta.SessionID, id)
	}
	if meta.ProfileID != "atb.profile.policy_decision" {
		t.Fatalf("ProfileID = %q, want policy_decision profile", meta.ProfileID)
	}
	if meta.EventCount != len(events) {
		t.Fatalf("EventCount = %d, want %d", meta.EventCount, len(events))
	}
	if meta.CreatedAt != fixedNow {
		t.Fatalf("CreatedAt = %v, want %v", meta.CreatedAt, fixedNow)
	}
	if meta.ClosedAt != fixedNow {
		t.Fatalf("ClosedAt = %v, want %v", meta.ClosedAt, fixedNow)
	}
	if meta.Path == "" {
		t.Fatal("expected non-empty bundle path")
	}
	wantPath := filepath.Join(mgr.dataDir, "sessions", id.String(), "bundle.atb")
	if meta.Path != wantPath {
		t.Fatalf("Path = %q, want %q", meta.Path, wantPath)
	}
	if meta.HeadHash == "" || meta.HeadHash[:7] != "sha256:" {
		t.Fatalf("HeadHash = %q, want sha256: prefix", meta.HeadHash)
	}

	if err := mgr.AppendEvent(ctx, id, events[0]); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("AppendEvent after close: got %v, want ErrSessionClosed", err)
	}
	if _, err := mgr.CloseSession(ctx, id, CloseSessionOpts{}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("CloseSession twice: got %v, want ErrSessionClosed", err)
	}
}

func TestMemoryBundleManagerCustomBundlePath(t *testing.T) {
	mgr := NewMemoryBundleManager(t.TempDir())
	customPath := filepath.Join(t.TempDir(), "custom", "bundle.atb")

	id, err := mgr.OpenSession(context.Background(), OpenParams{
		ActorID:    "actor-1",
		PurposeTag: "demo",
		BundlePath: customPath,
	})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}

	meta, err := mgr.CloseSession(context.Background(), id, CloseSessionOpts{})
	if err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	if meta.Path != customPath {
		t.Fatalf("Path = %q, want %q", meta.Path, customPath)
	}
}

func TestMemoryBundleManagerShutdownClearsSessions(t *testing.T) {
	mgr := NewMemoryBundleManager(t.TempDir())
	id, err := mgr.OpenSession(context.Background(), OpenParams{ActorID: "actor-1"})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if mgr.ActiveSessionCount() != 1 {
		t.Fatalf("ActiveSessionCount = %d, want 1", mgr.ActiveSessionCount())
	}

	if err := mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if mgr.ActiveSessionCount() != 0 {
		t.Fatalf("ActiveSessionCount after shutdown = %d, want 0", mgr.ActiveSessionCount())
	}
	if err := mgr.AppendEvent(context.Background(), id, PendingEvent{EventType: "test"}); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("AppendEvent after shutdown: got %v, want ErrSessionNotFound", err)
	}
}
