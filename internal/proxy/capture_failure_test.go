// SPDX-License-Identifier: MIT
package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/acquisition"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/capturejournal"
	"github.com/pcguest/atb/internal/event"
)

// enableAndAppend sets up capture, commits one observation, and closes the
// journal handle (simulating a stopped collector) so the durable state can be
// mutated by a test.
func enableAndAppend(t *testing.T, bundlePath string) *BundleRecorder {
	t.Helper()
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-fail"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.capture.journal.Close()
	return r
}

func checkpointPathFor(bundlePath string) string {
	return acquisition.ResolveCheckpointPath(bundlePath, captureSourceSystem, filepath.Base(bundlePath))
}

func recordCount(t *testing.T, bundlePath string) int {
	t.Helper()
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return len(b.Records)
}

func TestFailureMissingCheckpointRebindsWithoutDuplicates(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)
	before := recordCount(t, bundlePath)

	if err := os.Remove(checkpointPathFor(bundlePath)); err != nil {
		t.Fatalf("remove checkpoint: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-fail"); err != nil {
		t.Fatalf("recover without checkpoint: %v", err)
	}
	defer r2.capture.journal.Close()
	if got := recordCount(t, bundlePath); got != before {
		t.Fatalf("records = %d, want %d (no duplicates on rebind)", got, before)
	}
}

func TestFailureCorruptCheckpointFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	if err := os.WriteFile(checkpointPathFor(bundlePath), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("corrupt checkpoint: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-fail"); err == nil {
		_ = r2.capture.journal.Close()
		t.Fatalf("expected corrupt checkpoint to fail closed")
	}
}

func TestFailureInteriorJournalCorruptionFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	jp := journalPath(bundlePath)
	data, err := os.ReadFile(jp)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	// Rewrite the first line as valid JSON with a wrong chain hash.
	firstEnd := 0
	for i, b := range data {
		if b == '\n' {
			firstEnd = i
			break
		}
	}
	var e capturejournal.Entry
	if err := json.Unmarshal(data[:firstEnd], &e); err != nil {
		t.Fatalf("unmarshal first entry: %v", err)
	}
	e.Hash = "deadbeef"
	line, _ := json.Marshal(e)
	corrupt := append(append(line, '\n'), data[firstEnd+1:]...)
	if err := os.WriteFile(jp, corrupt, 0o600); err != nil {
		t.Fatalf("write corrupt journal: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-fail"); err == nil {
		_ = r2.capture.journal.Close()
		t.Fatalf("expected interior journal corruption to fail closed")
	}
}

func TestFailureAdapterVersionMismatchFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	cpPath := checkpointPathFor(bundlePath)
	cp, err := acquisition.Load(cpPath)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	cp.AdapterVersion = "2.0.0"
	if err := cp.Save(cpPath); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	err = r2.EnableCapture("inc-fail")
	if err == nil {
		_ = r2.capture.journal.Close()
		t.Fatalf("expected adapter version mismatch to fail closed")
	}
	if !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("error = %v, want adapter mismatch", err)
	}
}

func TestFailureSourceMismatchFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	cpPath := checkpointPathFor(bundlePath)
	cp, err := acquisition.Load(cpPath)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	cp.SourceSystem = "other.source"
	if err := cp.Save(cpPath); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-fail"); err == nil {
		_ = r2.capture.journal.Close()
		t.Fatalf("expected source mismatch to fail closed")
	}
}

func TestFailureCheckpointAheadOfEvidenceFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	cpPath := checkpointPathFor(bundlePath)
	cp, err := acquisition.Load(cpPath)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	cp.BundleRecordCount = 999
	if err := cp.Save(cpPath); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-fail"); err == nil {
		_ = r2.capture.journal.Close()
		t.Fatalf("expected checkpoint-ahead-of-evidence to fail closed")
	}
}

// TestFailureBundleSaveUnavailableJournalsThenRecovers models a disk/permission
// failure: the journal append succeeds (durable) but the evidence commit fails.
// The observation must remain recoverable and the capture state must be honest.
func TestFailureBundleSaveUnavailableJournalsThenRecovers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission semantics differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory permissions")
	}
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-fail"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o750) }()

	// Make the bundle directory unwritable so the durable evidence save fails.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err == nil {
		t.Fatalf("expected commit to fail while bundle dir is unwritable")
	}
	_ = r.capture.journal.Close()
	if r.capture.state != captureDegraded && r.capture.state != captureRecoveryRequired {
		t.Fatalf("state = %s, want degraded/recovery_required", r.capture.state)
	}

	// Restore permissions and recover: the durable journal observation is
	// replayed into evidence.
	if err := os.Chmod(dir, 0o750); err != nil {
		t.Fatalf("chmod restore: %v", err)
	}
	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-fail"); err != nil {
		t.Fatalf("recover: %v", err)
	}
	defer r2.capture.journal.Close()
	if r2.capture.replayed != 1 {
		t.Fatalf("replayed = %d, want 1", r2.capture.replayed)
	}
}
