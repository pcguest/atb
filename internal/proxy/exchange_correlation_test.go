// SPDX-License-Identifier: MIT
package proxy

import (
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

// TestOverlappingExchangeCorrelationIsPerRequest verifies that two overlapping
// requests in the same session produce two atb.exchange.complete records that
// each reference their own request record, rather than both referencing the
// most recent request (the pre-fix session-global behaviour). The requests are
// interleaved by hand: both requests are captured before either response, and
// the responses complete in reverse order, which is the scheduling that a
// delayed first response produces in production. This is a deterministic
// ordering test; the released-binary before/after is captured in the clean-room
// recertification.
func TestOverlappingExchangeCorrelationIsPerRequest(t *testing.T) {
	host := "api.anthropic.com"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-corr"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	// Close the journal before the test ends: on Windows an open handle
	// prevents t.TempDir from removing the journal file.
	defer func() { _ = r.Close() }()
	p := captureProxyForTest(t, r, host)

	corrA, err := captureRequestForTest(t, p, host, []byte(failureExchangeReqBody))
	if err != nil {
		t.Fatalf("request A: %v", err)
	}
	corrB, err := captureRequestForTest(t, p, host, []byte(failureExchangeReqBody))
	if err != nil {
		t.Fatalf("request B: %v", err)
	}
	if corrA.requestEventID == "" || corrB.requestEventID == "" {
		t.Fatalf("expected non-empty request refs, got A=%q B=%q", corrA.requestEventID, corrB.requestEventID)
	}
	if corrA.requestEventID == corrB.requestEventID {
		t.Fatalf("overlapping requests share a request ref: %q", corrA.requestEventID)
	}

	// Responses complete in reverse order (B before A).
	if err := captureResponseForTest(t, p, host, []byte(failureExchangeReqBody), []byte(failureExchangeRespBody), corrB); err != nil {
		t.Fatalf("response B: %v", err)
	}
	if err := captureResponseForTest(t, p, host, []byte(failureExchangeReqBody), []byte(failureExchangeRespBody), corrA); err != nil {
		t.Fatalf("response A: %v", err)
	}

	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	refs := map[string]int{}
	for _, rec := range b.Records {
		if rec.Event.Type != TypeExchangeComplete {
			continue
		}
		data, _ := rec.Event.Data.(map[string]any)
		id, _ := data["request_event_id"].(string)
		refs[id]++
	}
	if refs[corrA.requestEventID] != 1 || refs[corrB.requestEventID] != 1 {
		t.Fatalf("exchange refs = %v, want one each for A=%q and B=%q", refs, corrA.requestEventID, corrB.requestEventID)
	}
}

// TestResponseAfterSessionCloseUsesFreshSession verifies that a response whose
// request session was closed during the request/response gap (for example the
// idle timer firing during a long upstream response) is not recorded against
// the closed session, which would append exchange accounting and events after
// the session's close record.
func TestResponseAfterSessionCloseUsesFreshSession(t *testing.T) {
	host := "api.anthropic.com"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-closed"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer func() { _ = r.Close() }()
	p := captureProxyForTest(t, r, host)

	corr, err := captureRequestForTest(t, p, host, []byte(failureExchangeReqBody))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	// Simulate the idle timer closing the session during a long gap.
	p.sessions.closeSession(corr.sess)
	if !corr.sess.isClosed() {
		t.Fatalf("session should be closed")
	}

	if err := captureResponseForTest(t, p, host, []byte(failureExchangeReqBody), []byte(failureExchangeRespBody), corr); err != nil {
		t.Fatalf("response: %v", err)
	}
	if corr.sess.ExchangeCount != 0 || corr.sess.TotalTokens != 0 {
		t.Fatalf("closed session received late accounting: count=%d tokens=%d",
			corr.sess.ExchangeCount, corr.sess.TotalTokens)
	}
	// The response must still be recorded, on a fresh session rather than the
	// closed one: dropping the response entirely would also leave the closed
	// session unchanged.
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	recorded := false
	for _, rec := range b.Records {
		if rec.Event.Type != TypeExchangeComplete {
			continue
		}
		data, _ := rec.Event.Data.(map[string]any)
		sid, _ := data["session_id"].(string)
		if sid == corr.sess.ID {
			t.Fatalf("exchange recorded against the closed session %q", sid)
		}
		if sid != "" {
			recorded = true
		}
	}
	if !recorded {
		t.Fatalf("no exchange recorded against a fresh session after the original closed")
	}
}
