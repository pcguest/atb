// SPDX-License-Identifier: MIT
package proxy

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TestReadCaptureStatusDisclosesJournalDegradation verifies that a durable
// journal degradation marker (from a journal write failure) is surfaced offline
// as degraded/known_gap, so a restart cannot report healthy over lost evidence.
func TestReadCaptureStatusDisclosesJournalDegradation(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	_ = enableAndAppend(t, bundlePath)

	marker := journalPath(bundlePath) + ".degraded"
	if err := os.WriteFile(marker, []byte("{\"degraded_at\":\"2026-10-08T00:00:00Z\",\"reason\":\"journal write failed: simulated\"}\n"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.CaptureState != string(captureDegraded) {
		t.Fatalf("capture_state = %s, want degraded", st.CaptureState)
	}
	if !st.KnownGap {
		t.Fatalf("known_gap = false, want true")
	}
}

// TestLostJournalMarkerSurvivesRestart verifies that a degradation marker beside
// a lost journal keeps the capture degraded across a restart, rather than being
// cleared and reported healthy.
func TestLostJournalMarkerSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	if err := os.MkdirAll(filepath.Dir(journalPath(bundlePath)), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(journalPath(bundlePath)+".degraded", []byte("{\"degraded_at\":\"2026-10-08T00:00:00Z\",\"reason\":\"journal lost\"}\n"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-lost"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer func() { _ = r.capture.journal.Close() }()
	if r.capture.state != captureDegraded || !r.capture.knownGap {
		t.Fatalf("state=%s knownGap=%v, want degraded/true", r.capture.state, r.capture.knownGap)
	}
}

// TestHandoffSurfacesJournalDegradation verifies the degradation reaches a
// portable handoff as a known gap, not just the status view.
func TestHandoffSurfacesJournalDegradation(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	_ = enableAndAppend(t, bundlePath)

	marker := journalPath(bundlePath) + ".degraded"
	if err := os.WriteFile(marker, []byte("{\"degraded_at\":\"2026-10-08T00:00:00Z\",\"reason\":\"journal write failed: simulated\"}\n"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	h, err := BuildHandoff(bundlePath, 0)
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	if h.CaptureState != string(captureDegraded) {
		t.Fatalf("handoff capture_state = %s, want degraded", h.CaptureState)
	}
	if len(h.KnownGaps) == 0 {
		t.Fatalf("handoff known_gaps empty, want the degradation disclosed")
	}
}

// TestRequestFailureClearsStaleRequestEventHash verifies that when a request
// commit fails, the session's previous request hash is cleared, so a later
// atb.exchange.complete cannot point at a stale or non-existent request record.
func TestRequestFailureClearsStaleRequestEventHash(t *testing.T) {
	skipUnlessDirPermissionEnforced(t)
	host := "api.anthropic.com"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-req"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o750) }()
	p := captureProxyForTest(t, r, host)

	threadKey := func() string {
		req, err := http.NewRequest(http.MethodPost, "http://"+host+"/v1/messages", bytes.NewReader([]byte(failureExchangeReqBody)))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		return ExtractThreadKey(host, req, []byte(failureExchangeReqBody))
	}()

	// First request commits and records a request hash.
	if err := captureRequestForTest(t, p, host, []byte(failureExchangeReqBody)); err != nil {
		t.Fatalf("first request should commit: %v", err)
	}
	sess := p.sessions.Resolve(threadKey)
	if sess.getLastRequestEventHash() == "" {
		t.Fatalf("expected a recorded request hash after a successful commit")
	}

	// Second request fails; the stale hash must be cleared.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := captureRequestForTest(t, p, host, []byte(failureExchangeReqBody)); err == nil {
		t.Fatalf("expected the second request commit to fail")
	}
	if got := sess.getLastRequestEventHash(); got != "" {
		t.Fatalf("stale request hash not cleared on failure: %q", got)
	}
}
