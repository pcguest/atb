// SPDX-License-Identifier: MIT
package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/acquisition"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/capturejournal"
	"github.com/pcguest/atb/internal/event"
)

func newCaptureEvent(typ, session string) *event.Event {
	return &event.Event{
		Type:      typ,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Data:      map[string]any{"session_id": session, "host": "api.openai.com"},
	}
}

func journalPath(bundlePath string) string {
	return filepath.Join(filepath.Dir(bundlePath), captureJournalDir, filepath.Base(bundlePath)+".journal.ndjson")
}

func TestCaptureCommitWritesJournalAndV3Bundle(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-test"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer r.capture.journal.Close()

	hash, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1"))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if hash == "" {
		t.Fatalf("empty record hash")
	}

	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if m := b.Manifest(); m == nil || m.Version != "3" {
		t.Fatalf("manifest = %+v, want v3", b.Manifest())
	}
	last := b.Records[len(b.Records)-1]
	acq := last.Event.Acquisition
	if acq == nil {
		t.Fatalf("last record has no acquisition provenance")
	}
	if acq.Mode != "live" || acq.SourceSystem != captureSourceSystem || acq.Adapter != captureAdapter {
		t.Fatalf("acquisition = %+v", acq)
	}
	if acq.SourceRecordID == "" || acq.SourceDigest == "" {
		t.Fatalf("acquisition missing identity/digest: %+v", acq)
	}
	if acq.Checkpoint == nil || acq.Checkpoint.Position != "1" {
		t.Fatalf("acquisition checkpoint = %+v", acq.Checkpoint)
	}

	// The journal holds one durable entry.
	j, err := capturejournal.Open(journalPath(bundlePath), filepath.Base(bundlePath))
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	defer j.Close()
	if j.Len() != 1 {
		t.Fatalf("journal len = %d, want 1", j.Len())
	}
}

func TestCaptureRecoversJournalEntryBeforeCommit(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")

	r1 := NewBundleRecorder(bundlePath, nil)
	if err := r1.EnableCapture("inc-test"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r1.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	_ = r1.capture.journal.Close()

	// Simulate a crash after the journal append but before the bundle commit.
	fabricateJournalEntry(t, bundlePath, event.TypeLLMResponse, map[string]any{"session_id": "s1"})

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-test"); err != nil {
		t.Fatalf("recover EnableCapture: %v", err)
	}
	defer r2.capture.journal.Close()
	if r2.capture.replayed != 1 {
		t.Fatalf("replayed = %d, want 1", r2.capture.replayed)
	}

	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if len(b.Records) != 3 { // manifest + request + recovered response
		t.Fatalf("records = %d, want 3", len(b.Records))
	}
}

func TestCaptureReplayIsIdempotentAfterCommitBeforeCheckpoint(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")

	r1 := NewBundleRecorder(bundlePath, nil)
	if err := r1.EnableCapture("inc-test"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r1.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	_ = r1.capture.journal.Close()

	// Simulate a crash after the bundle commit but before the checkpoint save:
	// the journal and bundle contain the observation, but the checkpoint lags.
	entry := fabricateJournalEntry(t, bundlePath, event.TypeLLMResponse, map[string]any{"session_id": "s1"})
	appendCommittedEvent(t, bundlePath, entry)

	// Roll the checkpoint back to position 1 to model the lagging checkpoint.
	cpPath := r1.capture.cm.CheckpointPath
	cp, err := acquisition.Load(cpPath)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	cp.Position = "1"
	if err := cp.Save(cpPath); err != nil {
		t.Fatalf("rewrite checkpoint: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-test"); err != nil {
		t.Fatalf("recover EnableCapture: %v", err)
	}
	defer r2.capture.journal.Close()
	if r2.capture.replayed != 0 {
		t.Fatalf("replayed = %d, want 0 (already committed)", r2.capture.replayed)
	}
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if len(b.Records) != 3 { // no duplicate appended
		t.Fatalf("records = %d, want 3", len(b.Records))
	}
}

func TestCaptureWrongIncarnationFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")

	r1 := NewBundleRecorder(bundlePath, nil)
	if err := r1.EnableCapture("inc-a"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r1.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r1.capture.journal.Close()

	r2 := NewBundleRecorder(bundlePath, nil)
	err := r2.EnableCapture("inc-b")
	if err == nil {
		_ = r2.capture.journal.Close()
		t.Fatalf("expected incarnation mismatch to fail closed")
	}
	if !strings.Contains(err.Error(), "incarnation") {
		t.Fatalf("error = %v, want incarnation mismatch", err)
	}
}

func TestCaptureRepairsTornJournalTail(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")

	r1 := NewBundleRecorder(bundlePath, nil)
	if err := r1.EnableCapture("inc-test"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r1.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r1.capture.journal.Close()

	// Torn trailing line.
	f, err := os.OpenFile(journalPath(bundlePath), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open journal append: %v", err)
	}
	if _, err := f.WriteString(`{"format_version":1,"position":2,"observation_id":"x:2"`); err != nil {
		t.Fatalf("write torn tail: %v", err)
	}
	_ = f.Close()

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-test"); err != nil {
		t.Fatalf("recover after torn tail: %v", err)
	}
	defer r2.capture.journal.Close()
	if !r2.capture.knownGap {
		t.Fatalf("expected a known gap after torn-tail repair")
	}
	if r2.capture.state != captureDegraded {
		t.Fatalf("state = %s, want degraded", r2.capture.state)
	}
}

func TestReadCaptureStatus(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-status"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.capture.journal.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.Integrity != "verified" {
		t.Fatalf("integrity = %s", st.Integrity)
	}
	if st.ProcessHealth != "unknown" {
		t.Fatalf("process_health = %s, want unknown offline", st.ProcessHealth)
	}
	if st.CaptureState != string(captureHealthy) {
		t.Fatalf("capture_state = %s, want healthy (%s)", st.CaptureState, st.Detail)
	}
	if st.SourceIncarnation != "inc-status" {
		t.Fatalf("incarnation = %s", st.SourceIncarnation)
	}
	if st.JournalLastPosition != 1 || st.CommittedPosition != 1 || st.JournalBacklog != 0 {
		t.Fatalf("journal positions = %d/%d/%d", st.JournalLastPosition, st.CommittedPosition, st.JournalBacklog)
	}
}

func TestReadCaptureStatusReportsRecoveryRequired(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-status"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.capture.journal.Close()

	// A durable observation that was never committed.
	fabricateJournalEntry(t, bundlePath, event.TypeLLMResponse, map[string]any{"session_id": "s1"})

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != string(captureRecoveryRequired) {
		t.Fatalf("capture_state = %s, want recovery_required (%s)", st.CaptureState, st.Detail)
	}
	if st.JournalBacklog != 1 {
		t.Fatalf("backlog = %d, want 1", st.JournalBacklog)
	}
}

func TestCapturePathSecretCanary(t *testing.T) {
	const canary = "sk-CANARYcapture9f3a"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-test"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	rec := RequestRecord{
		SessionID:  "s1",
		Host:       "api.openai.com",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		APIKey:     canary,
		RecordedAt: time.Now().UTC(),
	}
	ev, err := rec.ToEvent()
	if err != nil {
		t.Fatalf("ToEvent: %v", err)
	}
	// The API key is used for identity resolution only; it must never enter the
	// event, the journal, or the bundle.
	if _, err := r.AppendEventHash(ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.capture.journal.Close()

	for _, path := range []string{bundlePath, journalPath(bundlePath)} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(data), canary) {
			t.Fatalf("%s contains the credential canary", path)
		}
	}
}

func TestBuildHandoffReportsGapWhenDegraded(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-handoff"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.capture.journal.Close()

	// Corrupt the journal interior so status reports a degraded state.
	data, err := os.ReadFile(journalPath(bundlePath))
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	firstEnd := 0
	for i, b := range data {
		if b == '\n' {
			firstEnd = i
			break
		}
	}
	var e capturejournal.Entry
	if err := json.Unmarshal(data[:firstEnd], &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	e.Hash = "deadbeef"
	line, _ := json.Marshal(e)
	if err := os.WriteFile(journalPath(bundlePath), append(append(line, '\n'), data[firstEnd+1:]...), 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}

	h, err := BuildHandoff(bundlePath, 0)
	if err != nil {
		t.Fatalf("BuildHandoff: %v", err)
	}
	if h.CaptureState == string(captureHealthy) {
		t.Fatalf("handoff should not report healthy for a corrupted journal")
	}
	if len(h.KnownGaps) == 0 {
		t.Fatalf("handoff should disclose the gap")
	}
}

func TestBuildHandoffCarriesIdentityAndLimitations(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-handoff"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.capture.journal.Close()

	h, err := BuildHandoff(bundlePath, 0)
	if err != nil {
		t.Fatalf("BuildHandoff: %v", err)
	}
	if h.BundleHead == "" || h.RecordHash == "" {
		t.Fatalf("handoff missing identity: %+v", h)
	}
	if h.EvidenceReference == "" || !strings.HasPrefix(h.EvidenceReference, "atb://evidence/1/") {
		t.Fatalf("evidence reference = %q", h.EvidenceReference)
	}
	if h.VerificationState != "verified" {
		t.Fatalf("verification state = %q", h.VerificationState)
	}
	if len(h.Limitations) == 0 {
		t.Fatalf("handoff must state limitations")
	}
	if h.CaptureState != string(captureHealthy) {
		t.Fatalf("capture state = %q", h.CaptureState)
	}
}

func TestValidateCheckpointPrefixFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-test"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer r.capture.journal.Close()
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	head := b.Records[len(b.Records)-1].Hash

	// A checkpoint ahead of the evidence must fail closed.
	ahead := &acquisition.Checkpoint{BundleHeadHash: head, BundleRecordCount: len(b.Records) + 5}
	if err := validateCheckpointPrefix(ahead, b); err == nil {
		t.Fatalf("expected checkpoint-ahead-of-evidence to fail closed")
	}
	// A checkpoint naming a non-prefix head must fail closed.
	foreign := &acquisition.Checkpoint{BundleHeadHash: "deadbeef", BundleRecordCount: 1}
	if err := validateCheckpointPrefix(foreign, b); err == nil {
		t.Fatalf("expected foreign head to fail closed")
	}
	// A matching prefix (bundle ahead of checkpoint) is accepted.
	prefix := &acquisition.Checkpoint{BundleHeadHash: b.Records[0].Hash, BundleRecordCount: 1}
	if err := validateCheckpointPrefix(prefix, b); err != nil {
		t.Fatalf("matching prefix should be accepted: %v", err)
	}
	// Unbound/legacy checkpoint is accepted.
	if err := validateCheckpointPrefix(&acquisition.Checkpoint{}, b); err != nil {
		t.Fatalf("unbound checkpoint should be accepted: %v", err)
	}
}

// fabricateJournalEntry appends a valid journal entry directly, simulating an
// observation that was durable in the journal but never committed.
func fabricateJournalEntry(t *testing.T, bundlePath, typ string, data map[string]any) capturejournal.Entry {
	t.Helper()
	streamID := filepath.Base(bundlePath)
	j, err := capturejournal.Open(journalPath(bundlePath), streamID)
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	defer j.Close()
	db, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	payload, err := json.Marshal(observationPayload{Type: typ, Data: db, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	canonical, err := canonicalize.Marshal(payload)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	sum := sha256.Sum256(canonical)
	pos := j.LastPosition() + 1
	entry, err := j.Append(capturejournal.Entry{
		ObservationID:         fmt.Sprintf("%s:%d", streamID, pos),
		SourceSystem:          captureSourceSystem,
		RepresentationVersion: captureRepresentationVersion,
		RepresentationDigest:  hex.EncodeToString(sum[:]),
		ObservationType:       typ,
		Payload:               payload,
		ObservedAt:            time.Now().UTC().Format(time.RFC3339Nano),
		Adapter:               captureAdapter,
		AdapterVersion:        captureAdapterVersion,
	})
	if err != nil {
		t.Fatalf("fabricate journal append: %v", err)
	}
	return entry
}

func appendCommittedEvent(t *testing.T, bundlePath string, entry capturejournal.Entry) {
	t.Helper()
	b, _, err := loadBundleForCapture(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if err := appendMaterialised(b, entry, filepath.Base(bundlePath)); err != nil {
		t.Fatalf("append materialised: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
}
