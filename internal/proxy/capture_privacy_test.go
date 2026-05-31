// SPDX-License-Identifier: MIT
package proxy_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/pcguest/atb/internal/proxy"
)

func eventData(t *testing.T, data any) map[string]any {
	t.Helper()
	m, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("event data is %T, want map[string]any", data)
	}
	return m
}

// TestCaptureDigestsBodyByDefault verifies that, by default, a captured
// request records only a SHA-256 digest and byte length — never the raw body.
// This is what makes the always-on recorder safe to run: no prompts,
// completions, or PII are persisted into the tamper-evident bundle.
func TestCaptureDigestsBodyByDefault(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"my SSN is 123-45-6789"}]}`)
	rec := proxy.RequestRecord{
		SessionID: "sess-1",
		Host:      "api.openai.com",
		Method:    "POST",
		Path:      "/v1/chat/completions",
		Body:      body,
		// CaptureBody defaults to false.
	}
	ev, err := rec.ToEvent()
	if err != nil {
		t.Fatalf("ToEvent: %v", err)
	}
	data := eventData(t, ev.Data)

	if _, ok := data["body"]; ok {
		t.Error("raw body must not be recorded by default")
	}
	wantDigest := hex.EncodeToString(func() []byte { s := sha256.Sum256(body); return s[:] }())
	if got := data["body_sha256"]; got != wantDigest {
		t.Errorf("body_sha256 = %v, want %s", got, wantDigest)
	}
	if got := data["body_bytes"]; got != len(body) {
		t.Errorf("body_bytes = %v, want %d", got, len(body))
	}
}

// TestCaptureBodiesOptIn verifies the raw body is retained only when opted in.
func TestCaptureBodiesOptIn(t *testing.T) {
	body := []byte(`{"model":"claude","messages":[]}`)
	rec := proxy.ResponseRecord{
		SessionID:   "sess-2",
		Host:        "api.anthropic.com",
		Method:      "POST",
		Path:        "/v1/messages",
		StatusCode:  200,
		Body:        body,
		CaptureBody: true,
	}
	ev, err := rec.ToEvent()
	if err != nil {
		t.Fatalf("ToEvent: %v", err)
	}
	data := eventData(t, ev.Data)

	if got := data["body"]; got != string(body) {
		t.Errorf("body = %v, want raw body when CaptureBody=true", got)
	}
	if _, ok := data["body_sha256"]; !ok {
		t.Error("body_sha256 must still be recorded when bodies are captured")
	}
}

// TestScanHeadersRedactsSecrets verifies credential and session-secret headers
// are never recorded, while ordinary headers pass through.
func TestScanHeadersRedactsSecrets(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer sk-secret")
	h.Set("X-Api-Key", "sk-secret-2")
	h.Set("Proxy-Authorization", "Basic abc")
	h.Set("Cookie", "session=secret")
	h.Set("Content-Type", "application/json")
	h.Set("OpenAI-Organization", "org-public")

	got := proxy.ScanHeaders(h)

	for _, secret := range []string{"Authorization", "X-Api-Key", "Proxy-Authorization", "Cookie"} {
		if _, ok := got[secret]; ok {
			t.Errorf("sensitive header %q must be redacted", secret)
		}
	}
	if got["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got["Content-Type"])
	}
	if got["Openai-Organization"] != "org-public" && got["OpenAI-Organization"] != "org-public" {
		t.Errorf("non-sensitive header should pass through, got %v", got)
	}
}
