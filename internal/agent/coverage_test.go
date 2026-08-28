// SPDX-License-Identifier: MIT
package agent

import (
	"bytes"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

func TestCaptureJSONAndErrorHelpers(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	tests := []struct {
		name string
		body *http.Request
		want string
	}{
		{name: "nil body", body: &http.Request{}, want: "body is required"},
		{name: "invalid", body: mustRequest(t, "{"), want: "invalid JSON"},
		{name: "unknown", body: mustRequest(t, `{"name":"ok","extra":true}`), want: "unknown field"},
		{name: "multiple", body: mustRequest(t, `{"name":"one"} {}`), want: "multiple JSON values"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var dst payload
			err := decodeJSONBody(tc.body, &dst)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}
	var dst payload
	if err := decodeJSONBody(mustRequest(t, `{"name":"ok"}`), &dst); err != nil || dst.Name != "ok" {
		t.Fatalf("decoded=%+v err=%v", dst, err)
	}

	for _, tc := range []struct {
		err    error
		status int
		msg    string
	}{
		{err: ErrSessionNotFound, status: http.StatusNotFound, msg: "session not found"},
		{err: ErrSessionClosed, status: http.StatusConflict, msg: "session closed"},
		{err: errors.New("other"), status: http.StatusInternalServerError, msg: "internal error"},
	} {
		status, msg := agentErrorHTTP(tc.err)
		if status != tc.status || msg != tc.msg {
			t.Fatalf("agentErrorHTTP(%v)=%d,%q", tc.err, status, msg)
		}
	}
}

func mustRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func TestCaptureBundleMetadataHelpers(t *testing.T) {
	sessionID := SessionID("session-1")
	if got := resolvedBundlePath("/data", sessionID, " custom.atb "); got != "custom.atb" {
		t.Fatalf("override path=%q", got)
	}
	if got := resolvedBundlePath("/data", sessionID, ""); got != filepath.Join("/data", "sessions", "session-1", "bundle.atb") {
		t.Fatalf("default path=%q", got)
	}

	data, err := pendingEventData(PendingEvent{EventType: " custom.event "})
	if err != nil || data["source_event_type"] != "custom.event" || len(data) != 1 {
		t.Fatalf("empty payload data=%v err=%v", data, err)
	}
	data, err = pendingEventData(PendingEvent{EventType: "custom.event", Payload: `{"ok":true}`})
	if err != nil || data["payload"] == nil {
		t.Fatalf("JSON payload data=%v err=%v", data, err)
	}
	data, err = pendingEventData(PendingEvent{EventType: "custom.event", Payload: " raw "})
	if err != nil || data["payload_raw"] != " raw " {
		t.Fatalf("raw payload data=%v err=%v", data, err)
	}

	if nonManifestRecordCount(nil) != 0 {
		t.Fatal("nil bundle record count nonzero")
	}
	if !manifestCreatedAt(nil).IsZero() || !manifestCreatedAt(&bundle.Bundle{}).IsZero() {
		t.Fatal("empty bundle timestamp nonzero")
	}
	nonManifest := &bundle.Bundle{Records: []bundle.Record{{}}}
	if !manifestCreatedAt(nonManifest).IsZero() {
		t.Fatal("non-manifest timestamp nonzero")
	}
	b, err := bundle.New()
	if err != nil {
		t.Fatal(err)
	}
	b.Records[0].Event.Timestamp = "invalid"
	if !manifestCreatedAt(b).IsZero() {
		t.Fatal("invalid manifest timestamp nonzero")
	}
	b.Records[0].Event.Timestamp = "2026-06-30T01:02:03Z"
	if got := manifestCreatedAt(b); !got.Equal(time.Date(2026, 6, 30, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("manifest timestamp=%v", got)
	}
}
