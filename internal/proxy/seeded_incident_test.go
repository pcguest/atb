// SPDX-License-Identifier: MIT
package proxy

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// runSyntheticExchange drives one OpenAI-shaped exchange (request + response
// with a tool call) through the real forward/capture helpers.
func runSyntheticExchange(t *testing.T, rec *BundleRecorder, cfg ProxyConfig, toolName, args string) {
	t.Helper()
	reqBody := `{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"act"}]}`
	respBody := fmt.Sprintf(
		`{"choices":[{"message":{"tool_calls":[{"id":"call_1","function":{"name":%q,"arguments":%q}}]}}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
		toolName, args,
	)
	req, err := http.NewRequest(http.MethodPost, "http://api.openai.com/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody}
	if _, err := RoundTripFixture("api.openai.com", req, []byte(reqBody), resp, []byte(respBody), rec, cfg); err != nil {
		t.Fatalf("RoundTripFixture: %v", err)
	}
}

// TestSeededIncidentContinuousCapture generates a real ATB bundle through the
// intercept capture path, including a collector restart mid-incident and an
// unexpected consequential tool operation.
func TestSeededIncidentContinuousCapture(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "incident.atb")
	cfg := ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  bundlePath,
		TargetHosts: []string{"api.openai.com"},
	}

	// First collector run: capture scope + a benign exchange.
	r1 := NewBundleRecorder(bundlePath, nil)
	if err := r1.EnableCapture("inc-acme-prod"); err != nil {
		t.Fatalf("EnableCapture 1: %v", err)
	}
	if err := r1.AppendEvent(CaptureScopeEvent(cfg)); err != nil {
		t.Fatalf("capture scope: %v", err)
	}
	runSyntheticExchange(t, r1, cfg, "list_orders", `{"account":"acme"}`)
	_ = r1.capture.journal.Close() // collector stops

	// Second collector run (restart, same incarnation): recovery, then the
	// unexpected consequential operation.
	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-acme-prod"); err != nil {
		t.Fatalf("EnableCapture 2 (restart): %v", err)
	}
	runSyntheticExchange(t, r2, cfg, "delete_account", `{"account":"acme","confirm":true}`)
	_ = r2.capture.journal.Close()

	// The capture process is now unavailable; the evidence must stand alone.
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}
	if err := b.Verify(); err != nil {
		t.Fatalf("verify: %v", err)
	}

	var toolCalls []string
	sawScope := false
	nonManifest := 0
	for _, rec := range b.Records {
		if rec.Event.Type == bundle.ManifestEventType {
			continue
		}
		nonManifest++
		switch rec.Event.Type {
		case event.TypeCaptureScope:
			sawScope = true
		case event.TypeToolCall:
			if d, ok := rec.Event.Data.(map[string]any); ok {
				if name, ok := d["tool_name"].(string); ok {
					toolCalls = append(toolCalls, name)
				}
			}
		}
		acq := rec.Event.Acquisition
		if acq == nil {
			t.Fatalf("non-manifest record seq %d (%s) has no acquisition provenance", rec.Event.Sequence, rec.Event.Type)
		}
		if acq.Mode != "live" {
			t.Fatalf("acquisition mode = %q, want live", acq.Mode)
		}
		if acq.SourceDigest == "" || acq.SourceRecordID == "" {
			t.Fatalf("acquisition missing identity: %+v", acq)
		}
	}
	if !sawScope {
		t.Fatalf("capture scope attestation missing")
	}
	if nonManifest == 0 {
		t.Fatalf("no captured records")
	}
	if !contains(toolCalls, "list_orders") || !contains(toolCalls, "delete_account") {
		t.Fatalf("tool calls = %v, want list_orders and delete_account", toolCalls)
	}

	// Capture health is readable offline and reflects established continuity.
	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != string(captureHealthy) {
		t.Fatalf("capture state = %s (%s)", st.CaptureState, st.Detail)
	}
	if st.SourceIncarnation != "inc-acme-prod" {
		t.Fatalf("incarnation = %s", st.SourceIncarnation)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
