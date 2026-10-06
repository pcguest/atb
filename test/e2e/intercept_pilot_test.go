// SPDX-License-Identifier: MIT
// Package e2e exercises the continuous-capture path end to end using the
// production proxy engine and the real on-disk artefacts (journal, checkpoint,
// bundle), including a collector restart and a durable capture degradation.
package e2e

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/proxy"
)

func roundTrip(t *testing.T, rec *proxy.BundleRecorder, cfg proxy.ProxyConfig, host, path, reqBody, respBody string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://"+host+path, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer sk-CANARY-e2e-should-not-persist")
	req.Header.Set("Content-Type", "application/json")
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(respBody)),
	}
	if _, err := proxy.RoundTripFixture(host, req, []byte(reqBody), resp, []byte(respBody), rec, cfg); err != nil {
		t.Fatalf("RoundTripFixture: %v", err)
	}
}

// TestInterceptPilotIncident runs a benign operation, an unexpected
// consequential operation, a collector restart, and a torn-tail degradation
// with recovery, then asserts the evidence, provenance, and honest capture
// state.
func TestInterceptPilotIncident(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "incident.atb")
	cfg := proxy.ProxyConfig{
		ListenAddr:        "127.0.0.1:0",
		BundlePath:        bundlePath,
		TargetHosts:       []string{"api.openai.com", "api.anthropic.com"},
		SourceIncarnation: "pilot-e2e-2026-10",
	}

	rec := proxy.NewBundleRecorder(bundlePath, nil)
	if err := rec.EnableCapture("pilot-e2e-2026-10"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	roundTrip(t, rec, cfg, "api.openai.com", "/v1/chat/completions",
		`{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"list my orders"}]}`,
		`{"model":"gpt-4.1-mini","choices":[{"message":{"tool_calls":[{"function":{"name":"list_orders","arguments":"{}"}}]}}]}`)
	roundTrip(t, rec, cfg, "api.anthropic.com", "/v1/messages",
		`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"do the thing"}]}`,
		`{"model":"claude-3-5-sonnet","content":[{"type":"tool_use","name":"delete_account","input":{"account_id":"acct-123"}}]}`)
	_ = rec.Close()

	// Collector restart.
	rec2 := proxy.NewBundleRecorder(bundlePath, nil)
	if err := rec2.EnableCapture("pilot-e2e-2026-10"); err != nil {
		t.Fatalf("restart EnableCapture: %v", err)
	}
	roundTrip(t, rec2, cfg, "api.openai.com", "/v1/chat/completions",
		`{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"after restart"}]}`,
		`{"model":"gpt-4.1-mini"}`)
	_ = rec2.Close()

	// Durable degradation: torn journal tail, then recover.
	jpath := filepath.Join(filepath.Dir(bundlePath), ".atb", "capture", filepath.Base(bundlePath)+".journal.ndjson")
	f, err := os.OpenFile(jpath, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	if _, err := f.WriteString(`{"format_version":1,"position":999,"observation_id":"torn`); err != nil {
		t.Fatalf("write torn tail: %v", err)
	}
	_ = f.Close()
	rec3 := proxy.NewBundleRecorder(bundlePath, nil)
	if err := rec3.EnableCapture("pilot-e2e-2026-10"); err != nil {
		t.Fatalf("recovery EnableCapture: %v", err)
	}
	roundTrip(t, rec3, cfg, "api.openai.com", "/v1/chat/completions",
		`{"model":"gpt-4.1-mini","messages":[{"role":"user","content":"after recovery"}]}`,
		`{"model":"gpt-4.1-mini"}`)
	_ = rec3.Close()

	// Evidence verifies offline.
	b, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("load verified: %v", err)
	}
	consequential := -1
	for _, r := range b.Records {
		if r.Event.Type == "atb.tool.call" {
			if s, ok := r.Event.Data.(map[string]any); ok {
				if tn, _ := s["tool_name"].(string); tn == "delete_account" {
					consequential = r.Event.Sequence
				}
			}
		}
	}
	if consequential < 0 {
		t.Fatalf("consequential delete_account tool call not found in evidence")
	}

	// Capture state is honestly degraded (durable torn-tail repair).
	st, err := proxy.ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.CaptureState != "degraded" || !st.KnownGap {
		t.Fatalf("capture_state = %s known_gap = %t, want degraded/known", st.CaptureState, st.KnownGap)
	}
	if st.SourceIncarnation != "pilot-e2e-2026-10" {
		t.Fatalf("incarnation = %s", st.SourceIncarnation)
	}

	// The handoff carries exact identity and capture limitations.
	h, err := proxy.BuildHandoff(bundlePath, consequential)
	if err != nil {
		t.Fatalf("BuildHandoff: %v", err)
	}
	if !strings.Contains(h.EvidenceReference, "record="+h.RecordHash) {
		t.Fatalf("evidence_reference %q missing record hash", h.EvidenceReference)
	}
	if h.CaptureState != "degraded" || len(h.KnownGaps) == 0 {
		t.Fatalf("handoff did not carry the capture limitation: %+v", h)
	}

	// The credential canary is absent from every artefact.
	for _, p := range []string{bundlePath, jpath} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if strings.Contains(string(data), "sk-CANARY-e2e") {
			t.Fatalf("%s contains the credential canary", p)
		}
	}
}
