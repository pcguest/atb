// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/proxy"
)

func makeCaptureBundle(t *testing.T) string {
	t.Helper()
	bundlePath := filepath.Join(t.TempDir(), "capture.atb")
	r := proxy.NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-cli"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	ev := &event.Event{
		Type:      event.TypeLLMRequest,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Data:      map[string]any{"session_id": "s1"},
	}
	if err := r.AppendEvent(ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	return bundlePath
}

func TestRunCaptureStatusValidatesFormatAndBundle(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runCaptureStatus([]string{"--format", "yaml"}, &out, &errBuf); code != exitUserError {
		t.Fatalf("invalid format exit = %d, want %d", code, exitUserError)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCaptureStatus([]string{"--bundle", ""}, &out, &errBuf); code != exitUserError {
		t.Fatalf("blank bundle exit = %d, want %d", code, exitUserError)
	}
}

func TestRunCaptureStatusJSON(t *testing.T) {
	bundlePath := makeCaptureBundle(t)
	var out, errBuf bytes.Buffer
	if code := runCaptureStatus([]string{"-b=" + bundlePath, "--format=json"}, &out, &errBuf); code != exitSuccess {
		t.Fatalf("status exit = %d (stderr %s)", code, errBuf.String())
	}
	var st proxy.CaptureStatus
	if err := json.Unmarshal(out.Bytes(), &st); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if st.CaptureState != "healthy" {
		t.Fatalf("capture_state = %q", st.CaptureState)
	}
}

func TestRunCaptureHandoffAcceptsShortBundleAndValidates(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runCaptureHandoff([]string{"--bundle"}, &out, &errBuf); code != exitUserError {
		t.Fatalf("missing value exit = %d, want %d", code, exitUserError)
	}
	out.Reset()
	errBuf.Reset()
	// A nonexistent bundle with -b= must not be reported as an unknown argument.
	code := runCaptureHandoff([]string{"-b=/nonexistent/does-not-exist.atb"}, &out, &errBuf)
	if code == exitUserError || strings.Contains(errBuf.String(), "unknown argument") {
		t.Fatalf("handoff -b= rejected: code=%d stderr=%s", code, errBuf.String())
	}
}
