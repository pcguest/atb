// SPDX-License-Identifier: MIT
package retentionaudit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestAppendCreatesAndExtendsVerifiedOperationsBundle(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".atb", "operations.atb")
	at := time.Date(2026, 6, 15, 1, 2, 3, 0, time.UTC)
	if err := Append(path, event.TypeDataRetentionPolicySet, map[string]any{"days": 183}, at); err != nil {
		t.Fatalf("Append() first event error = %v", err)
	}
	if err := Append(path, event.TypeDataRetentionEnforced, map[string]any{"operation": "archive"}, at.Add(time.Second)); err != nil {
		t.Fatalf("Append() second event error = %v", err)
	}
	b, err := bundle.LoadVerified(path)
	if err != nil {
		t.Fatalf("LoadVerified() error = %v", err)
	}
	if len(b.Records) != 3 {
		t.Fatalf("record count = %d, want manifest plus two events", len(b.Records))
	}
	if got := b.Records[2].Event.Type; got != event.TypeDataRetentionEnforced {
		t.Fatalf("last event type = %q", got)
	}
}

func TestPathForBundle(t *testing.T) {
	if got := PathForBundle(filepath.Join("/tmp/project", "run.atb", "bundle.atb")); got != filepath.Join("/tmp/project", ".atb", "operations.atb") {
		t.Fatalf("conventional path = %q", got)
	}
	if got := PathForBundle(filepath.Join("/tmp/evidence", "case.atb")); got != filepath.Join("/tmp/evidence", ".atb", "operations.atb") {
		t.Fatalf("explicit path = %q", got)
	}
}
