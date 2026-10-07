// SPDX-License-Identifier: MIT
package proxy

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/capturejournal"
	"github.com/pcguest/atb/internal/event"
)

func TestBodyTooLargeErrorContract(t *testing.T) {
	e := &BodyTooLargeError{Limit: 10, Observed: 20}
	msg := e.Error()
	if !strings.Contains(msg, "10") || !strings.Contains(msg, "20") {
		t.Fatalf("message = %q, want limit and observed", msg)
	}
	if !errors.Is(e, ErrBodyTooLarge) {
		t.Fatal("BodyTooLargeError should unwrap to ErrBodyTooLarge")
	}
}

func TestCaptureHealthAndValidationBranches(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-cov"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	defer r.Close()

	if err := r.CaptureHealth(); err != nil {
		t.Fatalf("fresh health = %v, want nil", err)
	}
	r.capture.fail(errors.New("boom"))
	if err := r.CaptureHealth(); err == nil {
		t.Fatal("health after failure should be non-nil")
	}
	if r.capture.Health() == nil {
		t.Fatal("coordinator Health should be non-nil after failure")
	}

	// Every contract-mismatch branch of validateEntry must fail closed.
	mismatches := []capturejournal.Entry{
		{SourceSystem: "x"},
		{SourceSystem: captureSourceSystem, Adapter: "x"},
		{SourceSystem: captureSourceSystem, Adapter: captureAdapter, AdapterVersion: "x"},
		{SourceSystem: captureSourceSystem, Adapter: captureAdapter, AdapterVersion: captureAdapterVersion, RepresentationVersion: "x"},
		{SourceSystem: captureSourceSystem, Adapter: captureAdapter, AdapterVersion: captureAdapterVersion, RepresentationVersion: captureRepresentationVersion, SourceIncarnation: "other"},
		{SourceSystem: captureSourceSystem, Adapter: captureAdapter, AdapterVersion: captureAdapterVersion, RepresentationVersion: captureRepresentationVersion, SourceIncarnation: "inc-cov", RepresentationDigest: "deadbeef"},
	}
	for i, e := range mismatches {
		if err := r.capture.validateEntry(e); err == nil {
			t.Fatalf("mismatch case %d: validateEntry returned nil", i)
		}
	}
}

func TestBuildHandoffSequenceNotFound(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-h"); err != nil {
		t.Fatalf("EnableCapture: %v", err)
	}
	if _, err := r.AppendEventHash(newCaptureEvent(event.TypeLLMRequest, "s1")); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = r.Close()

	if _, err := BuildHandoff(bundlePath, 9999); err == nil {
		t.Fatal("BuildHandoff with an unknown sequence should error")
	}
	if _, err := BuildHandoff(filepath.Join(dir, "missing.atb"), 0); err == nil {
		t.Fatal("BuildHandoff on a missing bundle should error")
	}
}

func TestCommittedHeadEmptyAndLoadNew(t *testing.T) {
	if committedHead(nil) != "" {
		t.Fatal("committedHead(nil) should be empty")
	}

	dir := t.TempDir()
	b2, created, err := loadBundleForCapture(filepath.Join(dir, "new.atb"))
	if err != nil || !created || b2 == nil {
		t.Fatalf("loadBundleForCapture new = (%v, %v, %v)", b2, created, err)
	}
}

func TestReadCaptureStatusBundleNotFound(t *testing.T) {
	st, err := ReadCaptureStatus(filepath.Join(t.TempDir(), "nope.atb"))
	if err != nil {
		t.Fatalf("ReadCaptureStatus: %v", err)
	}
	if st.Integrity != "unknown" || st.Detail != "bundle not found" {
		t.Fatalf("status = %+v, want bundle-not-found", st)
	}
}

func TestWriteTunnelError(t *testing.T) {
	var buf strings.Builder
	if err := writeTunnelError(&buf, nil, 400, "bad request"); err != nil {
		t.Fatalf("writeTunnelError: %v", err)
	}
	if !strings.Contains(buf.String(), "bad request") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestEnableCaptureDoubleAndAfterClose(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "capture.atb")
	r := NewBundleRecorder(bundlePath, nil)
	if err := r.EnableCapture("inc-double"); err != nil {
		t.Fatalf("first EnableCapture: %v", err)
	}
	if err := r.EnableCapture("inc-double"); err != nil {
		t.Fatalf("second EnableCapture should be a no-op: %v", err)
	}
	_ = r.Close()

	// A re-enable after close reopens the journal for the same stream.
	r2 := NewBundleRecorder(bundlePath, nil)
	if err := r2.EnableCapture("inc-double"); err != nil {
		t.Fatalf("re-enable after close: %v", err)
	}
	_ = r2.Close()
}

func TestCaptureRejectionNoRecorder(t *testing.T) {
	f := &forwarder{}
	// Must not panic when the forwarder has no proxy/recorder/sessions.
	f.recordCaptureRejection("api.openai.com", nil, "response", "upstream_connect_error", 1, 0)
}
