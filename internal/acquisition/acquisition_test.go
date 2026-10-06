// SPDX-License-Identifier: MIT
package acquisition

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	capturepkg "github.com/pcguest/atb/internal/capture"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/hash"
)

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "acquisition-continuity", name)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return p
}

func importPass(t *testing.T, bundlePath, inputPath string, reconcile bool) *ImportResult {
	t.Helper()
	res, err := ImportChatlog(context.Background(), ImportOptions{
		Format:        capturepkg.FormatGenericJSONL,
		InputPath:     inputPath,
		BundlePath:    bundlePath,
		MaxInputBytes: 1 << 20,
		Reconcile:     reconcile,
	})
	if err != nil {
		t.Fatalf("ImportChatlog(%s, reconcile=%v): %v", filepath.Base(inputPath), reconcile, err)
	}
	return res
}

func countFindings(t *testing.T, bundlePath string) int {
	t.Helper()
	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	n := 0
	for _, r := range b.Records {
		if r.Event.Type == event.TypeAcquisitionFinding {
			n++
		}
	}
	return n
}

func recordCount(t *testing.T, bundlePath string) int {
	t.Helper()
	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	return len(b.Records)
}

// TestThreePassAcquisitionLifecycle is the flagship regression: a chatlog
// source changes between acquisitions and must yield exactly one bounded
// finding, with a fully idempotent re-import and no duplicate evidence.
func TestThreePassAcquisitionLifecycle(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "acq.atb")

	// PASS 1: A1 B1 => A NEW, B NEW
	p1 := importPass(t, bundlePath, fixturePath(t, "pass1.jsonl"), true)
	if p1.NewCount != 2 || p1.ChangedCount != 0 || p1.UnchangedCount != 0 {
		t.Fatalf("pass1 counts: new=%d changed=%d unchanged=%d, want 2/0/0", p1.NewCount, p1.ChangedCount, p1.UnchangedCount)
	}
	if got := countFindings(t, bundlePath); got != 0 {
		t.Fatalf("pass1 findings = %d, want 0", got)
	}

	// PASS 2: A1 B2 C1 => A UNCHANGED, B CHANGED, C NEW, exactly one finding
	p2 := importPass(t, bundlePath, fixturePath(t, "pass2.jsonl"), true)
	if p2.UnchangedCount != 1 || p2.ChangedCount != 1 || p2.NewCount != 1 {
		t.Fatalf("pass2 counts: unchanged=%d changed=%d new=%d, want 1/1/1", p2.UnchangedCount, p2.ChangedCount, p2.NewCount)
	}
	if got := countFindings(t, bundlePath); got != 1 {
		t.Fatalf("pass2 findings = %d, want exactly 1", got)
	}
	afterPass2 := recordCount(t, bundlePath)

	// PASS 3: A1 B2 C1 => all UNCHANGED, zero new evidence, zero new findings
	p3 := importPass(t, bundlePath, fixturePath(t, "pass3.jsonl"), true)
	if p3.UnchangedCount != 3 || p3.ChangedCount != 0 || p3.NewCount != 0 {
		t.Fatalf("pass3 counts: unchanged=%d changed=%d new=%d, want 3/0/0", p3.UnchangedCount, p3.ChangedCount, p3.NewCount)
	}
	if p3.EventsWritten != 0 {
		t.Fatalf("pass3 wrote %d events, want 0 (idempotent re-import)", p3.EventsWritten)
	}
	if got := countFindings(t, bundlePath); got != 1 {
		t.Fatalf("pass3 findings = %d, want still 1 (idempotent)", got)
	}
	if got := recordCount(t, bundlePath); got != afterPass2 {
		t.Fatalf("pass3 record count = %d, want %d (no duplicate evidence)", got, afterPass2)
	}

	// Integrity must verify after the whole lifecycle.
	if _, err := bundle.LoadVerified(bundlePath); err != nil {
		t.Fatalf("LoadVerified after three passes: %v", err)
	}

	// The single finding must be bounded and reference the changed source record.
	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var finding map[string]any
	for _, r := range b.Records {
		if r.Event.Type == event.TypeAcquisitionFinding {
			m, ok := r.Event.Data.(map[string]any)
			if !ok {
				t.Fatalf("finding data type %T", r.Event.Data)
			}
			finding = m
		}
	}
	if finding == nil {
		t.Fatal("no acquisition finding found")
	}
	if finding["finding_type"] != "source_record_changed" {
		t.Fatalf("finding_type = %v", finding["finding_type"])
	}
	if finding["source_system"] != "chatlog" {
		t.Fatalf("source_system = %v", finding["source_system"])
	}
	if finding["source_record_id"] != "r2" {
		t.Fatalf("source_record_id = %v, want r2", finding["source_record_id"])
	}
	if finding["previous_digest"] == finding["current_digest"] {
		t.Fatal("previous_digest == current_digest for a CHANGED record")
	}

	// Acquisition stream must reflect the real input path, not the capture-time
	// "stdin" placeholder.
	for _, r := range b.Records {
		if r.Event.Acquisition != nil && r.Event.Acquisition.Checkpoint != nil {
			if r.Event.Acquisition.Checkpoint.AcquisitionStream == "stdin" {
				t.Fatalf("record %d kept placeholder acquisition stream", r.Event.Sequence)
			}
		}
	}
}

func TestResolveCheckpointPathSanitizesStream(t *testing.T) {
	p := ResolveCheckpointPath("/tmp/bundles/b.atb", "chatlog", "../../etc/passwd")
	if strings.Contains(p, "..") {
		t.Fatalf("checkpoint path retains traversal: %s", p)
	}
	if strings.ContainsAny(filepath.Base(p), `/\`) {
		t.Fatalf("checkpoint basename contains a separator: %s", filepath.Base(p))
	}
	if !strings.Contains(filepath.ToSlash(p), CheckpointDir) {
		t.Fatalf("checkpoint not under %s: %s", CheckpointDir, p)
	}
	if p1, p2 := ResolveCheckpointPath("/tmp/b.atb", "chatlog", "a/x.jsonl"), ResolveCheckpointPath("/tmp/b.atb", "chatlog", "b/x.jsonl"); p1 == p2 {
		t.Fatalf("distinct streams collided: %s", p1)
	}
}

func TestReconcilerOutcomes(t *testing.T) {
	r := NewReconciler(nil)

	// Unknown identity (no stable id).
	out, err := r.Reconcile(event.SourceIdentity{}, "d1", "t1", nil)
	if err != nil || out.Result != ReconciliationUnknown {
		t.Fatalf("unknown: result=%v err=%v", out.Result, err)
	}

	// New.
	id := event.SourceIdentity{System: "chatlog", RecordID: "r1", Derived: true}
	out, err = r.Reconcile(id, "d1", "t1", nil)
	if err != nil || out.Result != ReconciliationNew {
		t.Fatalf("new: result=%v err=%v", out.Result, err)
	}

	// Unchanged.
	out, _ = r.Reconcile(id, "d1", "t2", nil)
	if out.Result != ReconciliationUnchanged {
		t.Fatalf("unchanged: result=%v", out.Result)
	}

	// Changed -> one finding.
	out, _ = r.Reconcile(id, "d2", "t3", nil)
	if out.Result != ReconciliationChanged {
		t.Fatalf("changed: result=%v", out.Result)
	}
	if out.Finding == nil || out.Finding.Flag != "source_record_changed" {
		t.Fatalf("changed: finding=%+v", out.Finding)
	}
	if out.Finding.PreviousDigest != "d1" || out.Finding.CurrentDigest != "d2" {
		t.Fatalf("changed: digests prev=%s cur=%s", out.Finding.PreviousDigest, out.Finding.CurrentDigest)
	}
}

func TestLoadFromBundleSeedsKnownRecords(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "seed.atb")
	importPass(t, bundlePath, fixturePath(t, "pass1.jsonl"), false)

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	r := NewReconciler(nil)
	if err := r.LoadFromBundle(recordsToInterfaces(b.Records)); err != nil {
		t.Fatalf("LoadFromBundle: %v", err)
	}
	known := r.GetKnownRecord(event.SourceIdentity{System: "chatlog", RecordID: "r1", Derived: true})
	if known == nil {
		t.Fatal("expected known record for r1 after loading bundle")
	}
	if known.Digest == "" {
		t.Fatal("known record has empty digest")
	}
}

func TestCheckpointRoundTripAndValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cp.json")

	cp := &Checkpoint{
		SourceSystem:      "chatlog",
		AcquisitionStream: "stream-a",
		Position:          "exchange:1",
		ObservedAt:        "2026-01-01T00:00:00Z",
		Adapter:           "atb.chatlog.generic-jsonl",
		AdapterVersion:    "1.0.0",
		SourceIdentity:    event.SourceIdentity{System: "chatlog", RecordID: "stream-a", Derived: true},
		ProcessedCount:    1,
	}
	if err := cp.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.FormatVersion != CheckpointFormatVersion {
		t.Fatalf("format version = %d", got.FormatVersion)
	}
	if err := got.Validate("chatlog", "stream-a", "atb.chatlog.generic-jsonl", "", ""); err != nil {
		t.Fatalf("validate matching: %v", err)
	}
	if err := got.Validate("chatlog", "stream-b", "atb.chatlog.generic-jsonl", "", ""); !errors.Is(err, ErrCheckpointSourceMismatch) {
		t.Fatalf("stream mismatch: %v", err)
	}
	if err := got.Validate("chatlog", "stream-a", "other-adapter", "", ""); !errors.Is(err, ErrCheckpointAdapterMismatch) {
		t.Fatalf("adapter mismatch: %v", err)
	}

	if _, err := Load(filepath.Join(dir, "missing.json")); !errors.Is(err, ErrCheckpointNotFound) {
		t.Fatalf("missing: %v", err)
	}

	corrupt := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(corrupt, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(corrupt); !errors.Is(err, ErrCheckpointCorrupted) {
		t.Fatalf("corrupt: %v", err)
	}

	future := filepath.Join(dir, "future.json")
	if err := os.WriteFile(future, []byte(`{"format_version":99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(future); !errors.Is(err, ErrCheckpointVersionUnsupported) {
		t.Fatalf("future version: %v", err)
	}
}

func TestCheckpointSourceMismatchRejectsContinuation(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "acq.atb")
	cpPath := filepath.Join(dir, "fixed-checkpoint.json")

	if _, err := ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: fixturePath(t, "pass1.jsonl"),
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath,
	}); err != nil {
		t.Fatalf("initial import: %v", err)
	}

	_, err := ImportChatlog(context.Background(), ImportOptions{
		Format: capturepkg.FormatGenericJSONL, InputPath: fixturePath(t, "pass2.jsonl"),
		BundlePath: bundlePath, MaxInputBytes: 1 << 20, CheckpointPath: cpPath, Continue: true,
	})
	if !errors.Is(err, ErrCheckpointSourceMismatch) {
		t.Fatalf("expected ErrCheckpointSourceMismatch, got %v", err)
	}
}

func TestTamperedAcquisitionBundleRejected(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "acq.atb")
	importPass(t, bundlePath, fixturePath(t, "pass1.jsonl"), false)

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(b.Records) < 2 {
		t.Fatalf("expected >1 record, got %d", len(b.Records))
	}
	b.Records[1].Event.Data = map[string]any{"tampered": true}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save tampered: %v", err)
	}
	if _, err := bundle.LoadVerified(bundlePath); !errors.Is(err, bundle.ErrTamper) {
		t.Fatalf("expected ErrTamper, got %v", err)
	}
}

// TestLegacyBundleWithoutAcquisition pins two properties of the manifest-version
// floor:
//
//  1. A bundle with no acquisition metadata still loads and verifies unchanged
//     under the current build (historical compatibility).
//  2. Reconciling an acquisition-bearing import onto that pre-v3 bundle is
//     refused. The acquisition envelope participates in the canonical hash, so
//     appending it to a bundle that declares manifest v1/v2 would produce a
//     mixed-version chain that a pre-acquisition reader silently mis-hashes
//     (a spurious tamper). Per the migration policy, such evidence must be
//     re-imported into a new bundle rather than appended in place.
func TestLegacyBundleWithoutAcquisition(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "legacy.atb")

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	if err := b.AppendWithOptions("dev.session", map[string]any{"note": "legacy"}, &bundle.AppendOptions{
		Timestamp: "2026-01-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("append legacy: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save legacy: %v", err)
	}

	// 1. The legacy bundle still verifies unchanged.
	if _, err := bundle.LoadVerified(bundlePath); err != nil {
		t.Fatalf("legacy LoadVerified: %v", err)
	}

	// Snapshot the bundle before the refused import so we can prove no mutation.
	beforeBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle before refused import: %v", err)
	}
	beforeCount := recordCount(t, bundlePath)

	// 2. Reconciling an acquisition-bearing import onto it is refused loudly.
	_, err = ImportChatlog(context.Background(), ImportOptions{
		Format:        capturepkg.FormatGenericJSONL,
		InputPath:     fixturePath(t, "pass1.jsonl"),
		BundlePath:    bundlePath,
		MaxInputBytes: 1 << 20,
		Reconcile:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "manifest version < 3") {
		t.Fatalf("legacy reconcile: want manifest-version-floor error, got %v", err)
	}

	// The refused operation must not have mutated the bundle on disk.
	afterBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle after refused import: %v", err)
	}
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatalf("refused import mutated bundle bytes: before %d bytes, after %d bytes", len(beforeBytes), len(afterBytes))
	}
	if got := recordCount(t, bundlePath); got != beforeCount {
		t.Fatalf("refused import changed record count: before %d, after %d", beforeCount, got)
	}
	if _, err := bundle.LoadVerified(bundlePath); err != nil {
		t.Fatalf("LoadVerified after refused legacy reconcile: %v", err)
	}
}

// TestLegacyBundleNoOpReconcileAllowed pins the other half of the
// manifest-version floor: the refusal is only justified when an
// acquisition-bearing record would actually be written. A pre-v3 bundle that
// already contains the imported source records (as written by a build that
// predates the v3 declaration) reconciles as UNCHANGED, writes nothing, and so
// must not be refused.
func TestLegacyBundleNoOpReconcileAllowed(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "v3.atb")

	// Produce a bundle whose records carry acquisition, using the current writer.
	importPass(t, srcPath, fixturePath(t, "pass1.jsonl"), true)
	src, err := bundle.Load(srcPath)
	if err != nil {
		t.Fatalf("load v3 bundle: %v", err)
	}

	// Rebuild the same records as a pre-v3 bundle: a v1 manifest plus identical
	// acquisition-bearing events. This models a bundle written before the v3
	// declaration, when the acquisition envelope already participated in the
	// canonical hash but the manifest still declared v1.
	legacy, err := bundle.New()
	if err != nil {
		t.Fatalf("new legacy bundle: %v", err)
	}
	for _, rec := range src.Records {
		if rec.Event.Type == bundle.ManifestEventType {
			continue
		}
		appendLegacyAcquisitionRecord(t, legacy, rec)
	}
	legacyPath := filepath.Join(dir, "legacy.atb")
	if err := legacy.Save(legacyPath); err != nil {
		t.Fatalf("save legacy bundle: %v", err)
	}
	if _, err := bundle.LoadVerified(legacyPath); err != nil {
		t.Fatalf("legacy LoadVerified: %v", err)
	}

	beforeBytes, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy before import: %v", err)
	}
	beforeCount := recordCount(t, legacyPath)

	// A reconcile that changes nothing must succeed and write nothing, even
	// though the bundle is pre-v3 and its records carry acquisition.
	res, err := ImportChatlog(context.Background(), ImportOptions{
		Format:        capturepkg.FormatGenericJSONL,
		InputPath:     fixturePath(t, "pass1.jsonl"),
		BundlePath:    legacyPath,
		MaxInputBytes: 1 << 20,
		Reconcile:     true,
	})
	if err != nil {
		t.Fatalf("no-op reconcile on pre-v3 bundle: %v", err)
	}
	if res.EventsWritten != 0 {
		t.Fatalf("no-op reconcile wrote %d events, want 0", res.EventsWritten)
	}
	if res.UnchangedCount != 2 {
		t.Fatalf("no-op reconcile unchanged = %d, want 2", res.UnchangedCount)
	}
	if got := recordCount(t, legacyPath); got != beforeCount {
		t.Fatalf("no-op reconcile changed record count: before %d, after %d", beforeCount, got)
	}
	if _, err := bundle.LoadVerified(legacyPath); err != nil {
		t.Fatalf("LoadVerified after no-op reconcile: %v", err)
	}
	afterBytes, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy after import: %v", err)
	}
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatal("no-op reconcile mutated the pre-v3 bundle bytes")
	}
}

// TestLegacyBundleChangedReconcileRefused pins the complement of
// TestLegacyBundleNoOpReconcileAllowed: when a reconcile against a pre-v3
// bundle WOULD write an acquisition-bearing record (a changed source record or
// a new one), the manifest floor is enforced and the operation is refused
// rather than silently writing v3 canonical semantics into a legacy bundle.
func TestLegacyBundleChangedReconcileRefused(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "v3.atb")

	// Reconstruct pass1 (A1, B1) as a pre-v3 bundle carrying acquisition.
	importPass(t, srcPath, fixturePath(t, "pass1.jsonl"), true)
	src, err := bundle.Load(srcPath)
	if err != nil {
		t.Fatalf("load v3 bundle: %v", err)
	}

	legacy, err := bundle.New()
	if err != nil {
		t.Fatalf("new legacy bundle: %v", err)
	}
	for _, rec := range src.Records {
		if rec.Event.Type == bundle.ManifestEventType {
			continue
		}
		appendLegacyAcquisitionRecord(t, legacy, rec)
	}
	legacyPath := filepath.Join(dir, "legacy.atb")
	if err := legacy.Save(legacyPath); err != nil {
		t.Fatalf("save legacy bundle: %v", err)
	}
	if _, err := bundle.LoadVerified(legacyPath); err != nil {
		t.Fatalf("legacy LoadVerified: %v", err)
	}

	beforeBytes, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy before import: %v", err)
	}
	beforeCount := recordCount(t, legacyPath)

	// pass2 changes B1->B2 and adds C1, so this reconcile must write records.
	_, err = ImportChatlog(context.Background(), ImportOptions{
		Format:        capturepkg.FormatGenericJSONL,
		InputPath:     fixturePath(t, "pass2.jsonl"),
		BundlePath:    legacyPath,
		MaxInputBytes: 1 << 20,
		Reconcile:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "manifest version < 3") {
		t.Fatalf("changed reconcile on pre-v3 bundle: want manifest-version-floor error, got %v", err)
	}

	afterBytes, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy after refused import: %v", err)
	}
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatal("refused changed reconcile mutated the pre-v3 bundle bytes")
	}
	if got := recordCount(t, legacyPath); got != beforeCount {
		t.Fatalf("refused changed reconcile changed record count: before %d, after %d", beforeCount, got)
	}
	if _, err := bundle.LoadVerified(legacyPath); err != nil {
		t.Fatalf("LoadVerified after refused changed reconcile: %v", err)
	}
}

// appendLegacyAcquisitionRecord copies one acquisition-bearing event onto a
// pre-v3 bundle without passing through the bundle-layer floor (which refuses
// acquisition on a v1/v2 manifest), reconstructing the exact on-disk shape a
// pre-v3 writer produced: event data plus acquisition, hashed under the
// canonical profile of the time.
func appendLegacyAcquisitionRecord(t *testing.T, b *bundle.Bundle, src bundle.Record) {
	t.Helper()
	e := src.Event
	if err := b.AppendWithOptions(e.Type, e.Data, &bundle.AppendOptions{
		ActorID:      e.ActorID,
		OrgID:        e.OrgID,
		WorkspaceID:  e.WorkspaceID,
		Timestamp:    e.Timestamp,
		TraceID:      e.TraceID,
		SpanID:       e.SpanID,
		ParentSpanID: e.ParentSpanID,
	}); err != nil {
		t.Fatalf("append legacy event %s: %v", e.Type, err)
	}
	rec := &b.Records[len(b.Records)-1]
	rec.Event.Acquisition = e.Acquisition
	h, err := hash.Compute(rec.Event)
	if err != nil {
		t.Fatalf("hash legacy event %s: %v", e.Type, err)
	}
	rec.Hash = h
}
