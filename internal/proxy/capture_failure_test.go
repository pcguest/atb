// SPDX-License-Identifier: MIT
package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
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

// captureProxyForTest builds a proxy wired to recorder for driving the real
// request/response capture path directly.
func captureProxyForTest(t *testing.T, recorder *BundleRecorder, host string) *Proxy {
	t.Helper()
	cfg := ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  recorder.path,
		TargetHosts: []string{host},
	}
	p, err := NewProxy(cfg, LoggingHandler{}, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	p.recorder = recorder
	p.sessions = NewSessionManager(func(sess *Session) error {
		return recorder.AppendSessionClose(sess)
	})
	return p
}

func captureRequestForTest(t *testing.T, p *Proxy, host string, reqBody []byte) error {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "http://"+host+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return (&forwarder{proxy: p}).captureRequest(host, req, reqBody)
}

func captureResponseForTest(t *testing.T, p *Proxy, host string, reqBody, respBody []byte) error {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "http://"+host+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody}
	return (&forwarder{proxy: p}).captureResponse(host, req, resp, respBody)
}

func journalObservationTypes(j *capturejournal.Journal) map[string]int {
	out := map[string]int{}
	for _, e := range j.Entries() {
		out[e.ObservationType]++
	}
	return out
}

func bundleObservationTypes(t *testing.T, bundlePath string) map[string]int {
	t.Helper()
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	out := map[string]int{}
	for _, rec := range b.Records {
		out[rec.Event.Type]++
	}
	return out
}

func duplicateSourceRecordIDs(t *testing.T, bundlePath string) []string {
	t.Helper()
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	seen := map[string]bool{}
	var dups []string
	for _, rec := range b.Records {
		acq := rec.Event.Acquisition
		if acq == nil || acq.SourceRecordID == "" {
			continue
		}
		if seen[acq.SourceRecordID] {
			dups = append(dups, acq.SourceRecordID)
		}
		seen[acq.SourceRecordID] = true
	}
	return dups
}

const failureExchangeReqBody = `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":[` +
	`{"type":"tool_result","tool_use_id":"toolu_42","is_error":true,"content":"database unreachable"}` +
	`]}]}`

const failureExchangeRespBody = `{"model":"claude-3-5-sonnet","content":[` +
	`{"type":"text","text":"acting"},` +
	`{"type":"tool_use","name":"delete_user_records","input":{"id":"synthetic"}}` +
	`]}`

var failureExchangeWants = []string{
	event.TypeLLMRequest,
	event.TypeAIActionError,
	event.TypeLLMResponse,
	TypeExchangeComplete,
	event.TypeToolCall,
}

func skipUnlessDirPermissionEnforced(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("directory permission semantics differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory permissions")
	}
}

// TestFailureExchangeCommitFailureJournalsAllObservationsAndRecovers reproduces
// the clean-room P1: a bundle-save failure (permission/disk) during a
// multi-event exchange must not leave derived observations (tool call, exchange
// complete, action error) unjournalled. Every observation the proxy observed
// must remain durable and be recovered, and the recovered state must not claim
// completeness over silently dropped evidence.
func TestFailureExchangeCommitFailureJournalsAllObservationsAndRecovers(t *testing.T) {
	skipUnlessDirPermissionEnforced(t)
	host := "api.anthropic.com"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-exch"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o750) }()
	p := captureProxyForTest(t, r, host)

	// Every commit fails while the bundle directory is unwritable.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	reqErr := captureRequestForTest(t, p, host, []byte(failureExchangeReqBody))
	respErr := captureResponseForTest(t, p, host, []byte(failureExchangeReqBody), []byte(failureExchangeRespBody))
	if reqErr == nil || respErr == nil {
		t.Fatalf("expected commit failures, got reqErr=%v respErr=%v", reqErr, respErr)
	}

	// The journal must hold every observation for the exchange, not just the
	// primary request/response.
	jt := journalObservationTypes(r.capture.journal)
	for _, want := range failureExchangeWants {
		if jt[want] == 0 {
			t.Errorf("journal missing %s; have %v", want, jt)
		}
	}
	_ = r.capture.journal.Close()

	if err := os.Chmod(dir, 0o750); err != nil {
		t.Fatalf("chmod restore: %v", err)
	}
	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-exch"); err != nil {
		t.Fatalf("recover: %v", err)
	}
	defer r2.capture.journal.Close()
	if r2.capture.state != captureHealthy {
		t.Fatalf("state = %s, want healthy after full recovery", r2.capture.state)
	}

	bt := bundleObservationTypes(t, bundlePath)
	for _, want := range failureExchangeWants {
		if bt[want] == 0 {
			t.Errorf("recovered bundle missing %s; have %v", want, bt)
		}
		if bt[want] > 1 {
			t.Errorf("recovered bundle has %d %s records, want 1", bt[want], want)
		}
	}
	if dups := duplicateSourceRecordIDs(t, bundlePath); len(dups) > 0 {
		t.Fatalf("duplicate source record ids after recovery: %v", dups)
	}
}

// TestFailureExchangePartialCommitBoundaryRecoversDerivedObservations covers the
// boundary where the request commits successfully but the response-side commit
// fails: the derived exchange/tool observations must still be recovered exactly
// once.
func TestFailureExchangePartialCommitBoundaryRecoversDerivedObservations(t *testing.T) {
	skipUnlessDirPermissionEnforced(t)
	host := "api.anthropic.com"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-exch"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o750) }()
	p := captureProxyForTest(t, r, host)

	if err := captureRequestForTest(t, p, host, []byte(failureExchangeReqBody)); err != nil {
		t.Fatalf("request commit should succeed while writable: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := captureResponseForTest(t, p, host, []byte(failureExchangeReqBody), []byte(failureExchangeRespBody)); err == nil {
		t.Fatalf("expected response-side commit failure")
	}
	_ = r.capture.journal.Close()

	if err := os.Chmod(dir, 0o750); err != nil {
		t.Fatalf("chmod restore: %v", err)
	}
	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-exch"); err != nil {
		t.Fatalf("recover: %v", err)
	}
	defer r2.capture.journal.Close()

	bt := bundleObservationTypes(t, bundlePath)
	for _, want := range failureExchangeWants {
		if bt[want] != 1 {
			t.Errorf("recovered bundle has %d %s records, want 1; have %v", bt[want], want, bt)
		}
	}
	if dups := duplicateSourceRecordIDs(t, bundlePath); len(dups) > 0 {
		t.Fatalf("duplicate source record ids after recovery: %v", dups)
	}
}
