// SPDX-License-Identifier: MIT
package acquisition

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	capturepkg "github.com/pcguest/atb/internal/capture"
)

// A checkpoint bound to a committed bundle head must fail closed when the
// loaded bundle head (or record count) differs. A legacy unbound checkpoint
// (empty head) is accepted for backward compatibility.
func TestCheckpointBundleBindingFailsClosed(t *testing.T) {
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "cp.json")
	head := strings.Repeat("a", 64)

	cp := &Checkpoint{
		FormatVersion:     CheckpointFormatVersion,
		SourceSystem:      "chatlog",
		AcquisitionStream: "src",
		Adapter:           "atb.chatlog.generic-jsonl",
	}
	if err := cp.ValidateBundle(head, 3); err != nil {
		t.Fatalf("unbound (legacy) checkpoint must validate: %v", err)
	}

	cp.BindBundle(head, 3)
	if err := cp.Save(cpPath); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(cpPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loaded.ValidateBundle(head, 3); err != nil {
		t.Fatalf("matching head must validate: %v", err)
	}
	if err := loaded.ValidateBundle(strings.Repeat("b", 64), 3); !errors.Is(err, ErrCheckpointBundleMismatch) {
		t.Fatalf("different head must fail closed, got %v", err)
	}
	if err := loaded.ValidateBundle(head, 4); !errors.Is(err, ErrCheckpointBundleMismatch) {
		t.Fatalf("different record count must fail closed, got %v", err)
	}
}

// Continue must reconcile a re-import of the same source even when the caller
// sets Continue without Reconcile. Previously the library re-appended every
// record (silent duplicate evidence); only the CLI forced reconcile.
func TestLibraryContinueDeduplicatesWithoutReconcileFlag(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "acq.atb")
	cpPath := filepath.Join(dir, "cp.json")
	input := fixturePath(t, "pass1.jsonl")

	first, err := ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: input,
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath,
	})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.EventsWritten == 0 {
		t.Fatal("first import wrote no events")
	}

	second, err := ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: input,
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath, Continue: true,
	})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.EventsWritten != 0 {
		t.Fatalf("Continue=true must deduplicate a re-import of the same source: wrote %d", second.EventsWritten)
	}
	if second.UnchangedCount == 0 {
		t.Fatalf("expected unchanged records on re-import, got %+v", second)
	}
}

// A checkpoint whose bound head no longer matches the committed bundle head
// must abort continuation instead of silently resuming.
func TestContinueRejectsCheckpointBundleHeadMismatch(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "acq.atb")
	cpPath := filepath.Join(dir, "cp.json")
	input := fixturePath(t, "pass1.jsonl")

	if _, err := ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: input,
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath,
	}); err != nil {
		t.Fatalf("first import: %v", err)
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	cp, err := Load(cpPath)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if cp.BundleHeadHash == "" {
		t.Fatal("checkpoint was not bound to a committed bundle head")
	}
	cp.BindBundle(strings.Repeat("f", 64), len(b.Records))
	if err := cp.Save(cpPath); err != nil {
		t.Fatalf("save rebound checkpoint: %v", err)
	}

	_, err = ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: input,
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath, Continue: true,
	})
	if !errors.Is(err, ErrCheckpointBundleMismatch) {
		t.Fatalf("expected ErrCheckpointBundleMismatch, got %v", err)
	}
}

// A bound checkpoint must fail closed even when the bundle file is gone:
// otherwise a surviving checkpoint would be silently rebound to a new empty
// bundle. Unbound (first-run) checkpoints remain acceptable.
func TestContinueRejectsBoundCheckpointWhenBundleMissing(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "acq.atb")
	cpPath := filepath.Join(dir, "cp.json")
	input := fixturePath(t, "pass1.jsonl")

	if _, err := ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: input,
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath,
	}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	if err := os.Remove(bundlePath); err != nil {
		t.Fatalf("remove bundle: %v", err)
	}

	_, err := ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: input,
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath, Continue: true,
	})
	if !errors.Is(err, ErrCheckpointBundleMismatch) {
		t.Fatalf("expected ErrCheckpointBundleMismatch when a bound checkpoint outlives its bundle, got %v", err)
	}
}
