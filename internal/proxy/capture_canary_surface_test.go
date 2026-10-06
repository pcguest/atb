// SPDX-License-Identifier: MIT
package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCaptureCanaryAbsentAcrossSurfaces locks the default privacy contract: a
// synthetic secret in a body, a tool argument, or a credential field must not
// appear in the journal, the bundle, capture status, or the handoff.
func TestCaptureCanaryAbsentAcrossSurfaces(t *testing.T) {
	const canary = "sk-CANARYsurface1234567890"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-canary"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer r.Close()

	reqBody := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"` + canary + `"}]}`)
	rec := RequestRecord{
		SessionID: "s1", Host: "api.openai.com", Method: "POST",
		Path: "/v1/chat/completions", APIKey: canary, Body: reqBody, RecordedAt: time.Now().UTC(),
	}
	ev, err := rec.ToEvent()
	if err != nil {
		t.Fatalf("request ToEvent: %v", err)
	}
	if _, err := r.AppendEventHash(ev); err != nil {
		t.Fatalf("append request: %v", err)
	}

	respBody := []byte(`{"model":"gpt-4o","content":[{"type":"tool_use","name":"transfer","input":{"account":"` + canary + `"}}]}`)
	rrec := ResponseRecord{
		SessionID: "s1", Host: "api.openai.com", Method: "POST",
		Path: "/v1/messages", StatusCode: 200, Body: respBody, RecordedAt: time.Now().UTC(),
	}
	rev, err := rrec.ToEvent()
	if err != nil {
		t.Fatalf("response ToEvent: %v", err)
	}
	if _, err := r.AppendEventHash(rev); err != nil {
		t.Fatalf("append response: %v", err)
	}
	_ = r.Close()

	st, err := ReadCaptureStatus(bundlePath)
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	stJSON, _ := json.Marshal(st)
	h, err := BuildHandoff(bundlePath, 0)
	if err != nil {
		t.Fatalf("BuildHandoff: %v", err)
	}
	hJSON, _ := json.Marshal(h)

	surfaces := map[string][]byte{"status": stJSON, "handoff": hJSON}
	for _, p := range []string{bundlePath, journalPath(bundlePath)} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		surfaces[p] = data
	}
	for name, data := range surfaces {
		if strings.Contains(string(data), canary) {
			t.Fatalf("%s contains the credential canary in default mode", name)
		}
	}
}

// TestCaptureRawBodyModeChangesBoundary proves --capture-bodies intentionally
// widens the privacy boundary: the raw body (and therefore the canary) is
// retained only when opted in.
func TestCaptureRawBodyModeChangesBoundary(t *testing.T) {
	const canary = "sk-CANARYrawbody1234567890"
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-raw"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer r.Close()

	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"` + canary + `"}]}`)
	rec := RequestRecord{
		SessionID: "s1", Host: "api.openai.com", Method: "POST",
		Path: "/v1/chat/completions", Body: body, CaptureBody: true, RecordedAt: time.Now().UTC(),
	}
	ev, err := rec.ToEvent()
	if err != nil {
		t.Fatalf("ToEvent: %v", err)
	}
	if _, err := r.AppendEventHash(ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.Close()

	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if !strings.Contains(string(data), canary) {
		t.Fatalf("raw-body mode should retain the body canary in the bundle")
	}
}
