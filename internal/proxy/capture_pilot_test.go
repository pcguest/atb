// SPDX-License-Identifier: MIT
package proxy

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/capturejournal"
	"github.com/pcguest/atb/internal/event"
)

// TestCaptureJournalLossFailsClosed proves that deleting the journal while the
// bundle and checkpoint survive fails closed on restart and is reported as a
// possible unknown gap rather than silently resuming.
func TestCaptureJournalLossFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	if err := os.Remove(journalPath(bundlePath)); err != nil {
		t.Fatalf("remove journal: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	err := r2.EnableCapture("inc-fail")
	if err == nil {
		defer r2.capture.journal.Close()
		t.Fatalf("EnableCapture succeeded after journal loss; want fail closed")
	}

	st, serr := ReadCaptureStatus(bundlePath)
	if serr != nil {
		t.Fatalf("ReadCaptureStatus: %v", serr)
	}
	if st.CaptureState != string(captureDegraded) {
		t.Fatalf("capture_state = %s, want degraded (%s)", st.CaptureState, st.Detail)
	}
	if !st.PossibleUnknownGap {
		t.Fatalf("possible_unknown_gap = false, want true after journal loss")
	}
}

// TestCaptureOperationalStateLossFailsClosed proves that losing both the
// journal and the checkpoint while acquisition evidence remains fails closed
// and is reported honestly, rather than resuming with colliding identities.
func TestCaptureOperationalStateLossFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	if err := os.Remove(journalPath(bundlePath)); err != nil {
		t.Fatalf("remove journal: %v", err)
	}
	if err := os.Remove(checkpointPathFor(bundlePath)); err != nil {
		t.Fatalf("remove checkpoint: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	err := r2.EnableCapture("inc-new")
	if err == nil {
		defer r2.capture.journal.Close()
		t.Fatalf("EnableCapture succeeded after operational-state loss; want fail closed")
	}

	st, serr := ReadCaptureStatus(bundlePath)
	if serr != nil {
		t.Fatalf("ReadCaptureStatus: %v", serr)
	}
	if st.CaptureState == string(captureHealthy) {
		t.Fatalf("capture_state = healthy after state loss; want degraded/unknown (%s)", st.Detail)
	}
	if !st.PossibleUnknownGap {
		t.Fatalf("possible_unknown_gap = false, want true after state loss")
	}
}

// TestCapturePartialJournalLossFailsClosed proves that losing the checkpoint
// and part of the journal (but not all of it) fails closed, rather than
// resuming healthy over committed observations the journal can no longer
// account for.
func TestCapturePartialJournalLossFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-partial"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	_ = r.Close()

	if err := os.Remove(checkpointPathFor(bundlePath)); err != nil {
		t.Fatalf("remove checkpoint: %v", err)
	}
	data, err := os.ReadFile(journalPath(bundlePath))
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	idx := strings.IndexByte(string(data), '\n')
	if idx < 0 {
		t.Fatalf("journal has no complete line")
	}
	if err := os.WriteFile(journalPath(bundlePath), data[:idx+1], 0o600); err != nil {
		t.Fatalf("truncate journal: %v", err)
	}

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-partial"); err == nil {
		_ = r2.Close()
		t.Fatalf("EnableCapture succeeded with a partially lost journal; want fail closed")
	}
}

// TestCaptureMarkerIgnoredAfterJournalLoss proves a leftover repair marker does
// not mask journal loss as a repair.
func TestCaptureMarkerIgnoredAfterJournalLoss(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-marker"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.Close()

	// Torn tail, then repair (writes the marker).
	f, err := os.OpenFile(journalPath(bundlePath), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	if _, err := f.WriteString(`{"format_version":1,"position":999,"observation`); err != nil {
		t.Fatalf("write torn tail: %v", err)
	}
	_ = f.Close()
	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-marker"); err != nil {
		t.Fatalf("repair EnableCapture: %v", err)
	}
	_ = r2.Close()

	// Now the journal disappears but the marker remains.
	if err := os.Remove(journalPath(bundlePath)); err != nil {
		t.Fatalf("remove journal: %v", err)
	}
	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if !st.PossibleUnknownGap {
		t.Fatalf("journal loss with a stale marker should report possible_unknown_gap, got %s (%s)", st.CaptureState, st.Detail)
	}
	if strings.Contains(st.Detail, "torn tail") {
		t.Fatalf("journal loss was masked as a repair: %s", st.Detail)
	}
}

// TestCaptureStatusRejectsUnsupportedRepresentationVersion proves offline
// status rejects a journal entry with an unsupported representation version.
func TestCaptureStatusRejectsUnsupportedRepresentationVersion(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	j, err := capturejournal.Open(journalPath(bundlePath), filepath.Base(bundlePath))
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	payload, _ := json.Marshal(observationPayload{Type: event.TypeLLMResponse, Data: json.RawMessage(`{"session_id":"s1"}`)})
	digest, _ := digestRepresentation(payload)
	if _, err := j.Append(capturejournal.Entry{
		ObservationID:         "inc-fail/" + filepath.Base(bundlePath) + ":2",
		SourceSystem:          captureSourceSystem,
		SourceIncarnation:     "inc-fail",
		RepresentationVersion: "atb.intercept.event.v0",
		RepresentationDigest:  digest,
		ObservationType:       event.TypeLLMResponse,
		Payload:               payload,
		ObservedAt:            time.Now().UTC().Format(time.RFC3339Nano),
		Adapter:               captureAdapter,
		AdapterVersion:        captureAdapterVersion,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = j.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != string(captureDegraded) || !st.KnownGap {
		t.Fatalf("capture_state = %s known_gap = %t, want degraded/known", st.CaptureState, st.KnownGap)
	}
}

// TestCaptureStatusRejectsMixedIncarnations proves a journal mixing two source
// incarnations is not reported as continuous.
func TestCaptureStatusRejectsMixedIncarnations(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	j, err := capturejournal.Open(journalPath(bundlePath), filepath.Base(bundlePath))
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	payload, _ := json.Marshal(observationPayload{Type: event.TypeLLMResponse, Data: json.RawMessage(`{"session_id":"s1"}`)})
	digest, _ := digestRepresentation(payload)
	if _, err := j.Append(capturejournal.Entry{
		ObservationID:         "inc-other/" + filepath.Base(bundlePath) + ":2",
		SourceSystem:          captureSourceSystem,
		SourceIncarnation:     "inc-other",
		RepresentationVersion: captureRepresentationVersion,
		RepresentationDigest:  digest,
		ObservationType:       event.TypeLLMResponse,
		Payload:               payload,
		ObservedAt:            time.Now().UTC().Format(time.RFC3339Nano),
		Adapter:               captureAdapter,
		AdapterVersion:        captureAdapterVersion,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = j.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != string(captureDegraded) || !st.KnownGap {
		t.Fatalf("capture_state = %s known_gap = %t, want degraded/known", st.CaptureState, st.KnownGap)
	}
}

// TestCapturePathJournalsRejectionEvents proves source-unavailable and
// oversized-input rejections are journalled and committed with live
// acquisition provenance on the capture path, not silently dropped.
func TestCapturePathJournalsRejectionEvents(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	rec := NewBundleRecorder(bundlePath, nil)
	if err := rec.EnableCapture("inc-reject"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	cfg := ProxyConfig{ListenAddr: "127.0.0.1:0", BundlePath: bundlePath, TargetHosts: []string{"api.openai.com"}}
	p, err := NewProxy(cfg, LoggingHandler{}, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	p.recorder = rec
	p.sessions = NewSessionManager(rec.sessionCloseCallback)
	f := &forwarder{proxy: p}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	f.recordCaptureRejection("api.openai.com", req, "response", "upstream_connect_error", 1024, 0)
	f.recordCaptureRejection("api.openai.com", req, "request", "body_too_large", 1024, 2048)
	_ = rec.Close()

	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	rejections := 0
	for _, r := range b.Records {
		if r.Event.Type != event.TypeCaptureRejected {
			continue
		}
		rejections++
		if r.Event.Acquisition == nil || r.Event.Acquisition.Mode != "live" {
			t.Fatalf("rejection record missing live acquisition provenance: %+v", r.Event.Acquisition)
		}
	}
	if rejections != 2 {
		t.Fatalf("rejection records = %d, want 2", rejections)
	}
}

// TestCaptureStatusDetectsDigestMismatch proves a self-asserted representation
// digest that does not match its payload is reported as a known gap.
func TestCaptureStatusDetectsDigestMismatch(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	// Fabricate a validly-chained entry whose recorded digest is wrong.
	j, err := capturejournal.Open(journalPath(bundlePath), filepath.Base(bundlePath))
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	payload, _ := json.Marshal(observationPayload{Type: event.TypeLLMResponse, Data: json.RawMessage(`{"session_id":"s1"}`)})
	if _, err := j.Append(capturejournal.Entry{
		ObservationID:         "inc-fail/" + filepath.Base(bundlePath) + ":2",
		SourceSystem:          captureSourceSystem,
		SourceIncarnation:     "inc-fail",
		RepresentationVersion: captureRepresentationVersion,
		RepresentationDigest:  strings.Repeat("0", 64),
		ObservationType:       event.TypeLLMResponse,
		Payload:               payload,
		ObservedAt:            time.Now().UTC().Format(time.RFC3339Nano),
		Adapter:               captureAdapter,
		AdapterVersion:        captureAdapterVersion,
	}); err != nil {
		t.Fatalf("append mismatched entry: %v", err)
	}
	_ = j.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != string(captureDegraded) || !st.KnownGap {
		t.Fatalf("capture_state = %s known_gap = %t, want degraded/known (%s)", st.CaptureState, st.KnownGap, st.Detail)
	}
}

// TestCaptureStatusDetectsSubstitutedJournal proves a journal written by a
// different source/adapter is not reported as continuous.
func TestCaptureStatusDetectsSubstitutedJournal(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	enableAndAppend(t, bundlePath)

	j, err := capturejournal.Open(journalPath(bundlePath), filepath.Base(bundlePath))
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	payload, _ := json.Marshal(observationPayload{Type: event.TypeLLMResponse, Data: json.RawMessage(`{"session_id":"s1"}`)})
	digest, _ := digestRepresentation(payload)
	if _, err := j.Append(capturejournal.Entry{
		ObservationID:         "inc-fail/" + filepath.Base(bundlePath) + ":2",
		SourceSystem:          captureSourceSystem,
		SourceIncarnation:     "inc-fail",
		RepresentationVersion: captureRepresentationVersion,
		RepresentationDigest:  digest,
		ObservationType:       event.TypeLLMResponse,
		Payload:               payload,
		ObservedAt:            time.Now().UTC().Format(time.RFC3339Nano),
		Adapter:               "atb.evil",
		AdapterVersion:        captureAdapterVersion,
	}); err != nil {
		t.Fatalf("append substituted entry: %v", err)
	}
	_ = j.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != string(captureDegraded) || !st.KnownGap {
		t.Fatalf("capture_state = %s known_gap = %t, want degraded/known (%s)", st.CaptureState, st.KnownGap, st.Detail)
	}
}

// TestCapturePreservesSourceAndObservedTime proves source time and observing
// time are recorded separately and that recorded order (seq) is independent of
// source time, including backwards and identical source timestamps.
func TestCapturePreservesSourceAndObservedTime(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-time"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer r.Close()

	times := []string{
		"2030-01-01T00:00:00Z", // future vs observing clock
		"2020-01-01T00:00:00Z", // backwards
		"2020-01-01T00:00:00Z", // identical
	}
	for _, ts := range times {
		ev := newCaptureEvent(event.TypeLLMRequest, "s1")
		ev.Timestamp = ts
		if _, err := r.AppendEventHash(ev); err != nil {
			t.Fatalf("append %s: %v", ts, err)
		}
	}

	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var acqRecords []int
	for _, rec := range b.Records {
		if rec.Event.Acquisition == nil {
			continue
		}
		acqRecords = append(acqRecords, rec.Event.Sequence)
	}
	if len(acqRecords) != 3 {
		t.Fatalf("acquisition records = %d, want 3", len(acqRecords))
	}
	// Recorded order is strictly increasing and independent of source time.
	for i := 1; i < len(acqRecords); i++ {
		if acqRecords[i] <= acqRecords[i-1] {
			t.Fatalf("recorded order not increasing: %v", acqRecords)
		}
	}
	// Source timestamp is preserved verbatim and differs from acquired_at.
	last := b.Records[len(b.Records)-1]
	if last.Event.Acquisition.SourceTimestamp != "2020-01-01T00:00:00Z" {
		t.Fatalf("source_timestamp = %q, want preserved source time", last.Event.Acquisition.SourceTimestamp)
	}
	if last.Event.Acquisition.AcquiredAt == "" || last.Event.Acquisition.AcquiredAt == last.Event.Acquisition.SourceTimestamp {
		t.Fatalf("acquired_at = %q, want a distinct observing time", last.Event.Acquisition.AcquiredAt)
	}
}

// TestCaptureTornTailRepairIsDurablyVisible proves a repaired torn tail is
// still reported as a known gap by an offline status read after the repairing
// process has exited.
func TestCaptureTornTailRepairIsDurablyVisible(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-repair"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.Close()

	f, err := os.OpenFile(journalPath(bundlePath), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	if _, err := f.WriteString(`{"format_version":1,"position":999,"observation`); err != nil {
		t.Fatalf("write torn tail: %v", err)
	}
	_ = f.Close()

	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-repair"); err != nil {
		t.Fatalf("recover after torn tail: %v", err)
	}
	_ = r2.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != string(captureDegraded) || !st.KnownGap {
		t.Fatalf("capture_state = %s known_gap = %t, want degraded/known after a durable repair", st.CaptureState, st.KnownGap)
	}
}

// TestCaptureLateStartIsNotEstablished proves a configured-but-idle capture is
// reported as not established with no observation currency, rather than
// healthy, so an operator cannot infer full coverage before observation began.
func TestCaptureLateStartIsNotEstablished(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-idle"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	_ = r.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState == string(captureHealthy) {
		t.Fatalf("capture_state = healthy for an idle capture; want not_established")
	}
	if st.ObservationCurrency != "none" {
		t.Fatalf("observation_currency = %q, want none", st.ObservationCurrency)
	}
}
