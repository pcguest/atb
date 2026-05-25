// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
	"github.com/pcguest/atb/pkg/custody"
)

func TestBundleFileManagerLifecycle(t *testing.T) {
	fixedNow := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "agent")
	mgr := NewBundleFileManager(root)
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

	bundlePath := filepath.Join(root, "sessions", id.String(), "bundle.atb")
	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("bundle file after open: %v", err)
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
	if meta.EventCount != len(events) {
		t.Fatalf("EventCount = %d, want %d", meta.EventCount, len(events))
	}
	if meta.Path != bundlePath {
		t.Fatalf("Path = %q, want %q", meta.Path, bundlePath)
	}
	if meta.ClosedAt != fixedNow {
		t.Fatalf("ClosedAt = %v, want %v", meta.ClosedAt, fixedNow)
	}

	loaded, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(loaded.Records) < 1+len(events) {
		t.Fatalf("record count = %d, want at least %d", len(loaded.Records), 1+len(events))
	}

	head := custody.HeadHash(loaded)
	if meta.HeadHash != head {
		t.Fatalf("HeadHash = %q, want %q", meta.HeadHash, head)
	}

	report := verify.Verify(loaded, bundlePath, meta.ProfileID)
	if !report.Integrity.ChainValid {
		t.Fatalf("verify integrity: chain_valid=false error=%q", report.Integrity.Error)
	}

	last := loaded.Records[len(loaded.Records)-1]
	if last.Event.Type != agentRawEventType {
		t.Fatalf("last event type = %q, want %q", last.Event.Type, agentRawEventType)
	}

	if err := mgr.AppendEvent(ctx, id, events[0]); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("AppendEvent after close: got %v, want ErrSessionClosed", err)
	}
}

func TestBundleFileManagerOpenCreatesManifestOnlyBundle(t *testing.T) {
	mgr := NewBundleFileManager(t.TempDir())
	id, err := mgr.OpenSession(context.Background(), OpenParams{ActorID: "actor-1"})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}

	path := sessionBundlePath(mgr.dataDir, id)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat bundle: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty bundle file after open")
	}

	loaded, err := bundle.LoadVerified(path)
	if err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}
	if len(loaded.Records) != 1 {
		t.Fatalf("records = %d, want 1 manifest", len(loaded.Records))
	}
}

func TestBundleFileManagerCustomBundlePathResume(t *testing.T) {
	root := t.TempDir()
	mgr := NewBundleFileManager(root)
	customPath := filepath.Join(root, "custom", "bundle.atb")

	id, err := mgr.OpenSession(context.Background(), OpenParams{
		ActorID:    "actor-1",
		BundlePath: customPath,
	})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if err := mgr.AppendEvent(context.Background(), id, PendingEvent{
		EventType: "demo.event",
		Payload:   `{"k":1}`,
	}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if _, err := mgr.CloseSession(context.Background(), id, CloseSessionOpts{}); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	mgr2 := NewBundleFileManager(root)
	id2, err := mgr2.OpenSession(context.Background(), OpenParams{BundlePath: customPath})
	if err != nil {
		t.Fatalf("resume OpenSession: %v", err)
	}
	if err := mgr2.AppendEvent(context.Background(), id2, PendingEvent{
		EventType: "demo.event",
		Payload:   `{"k":2}`,
	}); err != nil {
		t.Fatalf("AppendEvent on resume: %v", err)
	}
	meta, err := mgr2.CloseSession(context.Background(), id2, CloseSessionOpts{})
	if err != nil {
		t.Fatalf("CloseSession on resume: %v", err)
	}
	if meta.EventCount != 2 {
		t.Fatalf("EventCount = %d, want 2", meta.EventCount)
	}
}

func TestBundleFileManagerShutdownClearsSessions(t *testing.T) {
	mgr := NewBundleFileManager(t.TempDir())
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
