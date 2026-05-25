// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"testing"
)

func TestRuntimeShutdownClearsBundleManager(t *testing.T) {
	cfg, err := LoadConfigFromEnv("1.11.0-test", func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	cfg.DataDir = t.TempDir()

	rt, err := NewRuntime(cfg, nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	mgr, ok := rt.bundleManager.(*BundleFileManager)
	if !ok {
		t.Fatalf("bundle manager type = %T, want *BundleFileManager", rt.bundleManager)
	}

	id, err := mgr.OpenSession(context.Background(), OpenParams{ActorID: "actor-1"})
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if mgr.ActiveSessionCount() != 1 {
		t.Fatalf("ActiveSessionCount = %d, want 1", mgr.ActiveSessionCount())
	}

	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if mgr.ActiveSessionCount() != 0 {
		t.Fatalf("ActiveSessionCount after shutdown = %d, want 0", mgr.ActiveSessionCount())
	}
	if err := mgr.AppendEvent(context.Background(), id, PendingEvent{EventType: "test"}); err != ErrSessionNotFound {
		t.Fatalf("AppendEvent after runtime shutdown: got %v, want ErrSessionNotFound", err)
	}
}
