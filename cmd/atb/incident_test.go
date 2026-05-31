// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

func writeIncidentBundle(t *testing.T) string {
	t.Helper()
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, event.TypeLLMRequest, map[string]any{
		"session_id": "sess-A", "host": "api.anthropic.com", "method": "POST", "path": "/v1/messages",
	})
	appendTestBundleEvent(t, b, event.TypeToolCall, map[string]any{
		"session_id": "sess-A", "tool_name": "wipe_db",
	})
	appendTestBundleEvent(t, b, event.TypeSessionClose, map[string]any{
		"session_id": "sess-A", "actor_id": "agent-x",
	})
	path := filepath.Join(t.TempDir(), "capture.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	return path
}

func TestRunIncidentReportMarkdown(t *testing.T) {
	path := writeIncidentBundle(t)
	var out, errBuf bytes.Buffer
	code := runIncident([]string{"report", "--bundle", path, "--session", "sess-A"}, &out, &errBuf)
	if code != exitSuccess {
		t.Fatalf("exit = %d, stderr=%s", code, errBuf.String())
	}
	got := out.String()
	if !strings.Contains(got, "sess-A") || !strings.Contains(got, "tool=wipe_db") {
		t.Errorf("unexpected report:\n%s", got)
	}
	if !strings.Contains(got, "tool_without_approval") {
		t.Errorf("expected anomaly in report:\n%s", got)
	}
}

func TestRunIncidentReportJSON(t *testing.T) {
	path := writeIncidentBundle(t)
	var out, errBuf bytes.Buffer
	code := runIncident([]string{"report", "--bundle", path, "--session=sess-A", "--format", "json"}, &out, &errBuf)
	if code != exitSuccess {
		t.Fatalf("exit = %d, stderr=%s", code, errBuf.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if payload["session_id"] != "sess-A" {
		t.Errorf("session_id = %v", payload["session_id"])
	}
}

func TestRunIncidentList(t *testing.T) {
	path := writeIncidentBundle(t)

	var out, errBuf bytes.Buffer
	if code := runIncident([]string{"list", "--bundle", path}, &out, &errBuf); code != exitSuccess {
		t.Fatalf("list exit = %d, stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "sess-A") {
		t.Errorf("list missing sess-A:\n%s", out.String())
	}

	var jOut, jErr bytes.Buffer
	if code := runIncident([]string{"list", "--bundle", path, "--format", "json"}, &jOut, &jErr); code != exitSuccess {
		t.Fatalf("list json exit = %d, stderr=%s", code, jErr.String())
	}
	var entries []map[string]any
	if err := json.Unmarshal(jOut.Bytes(), &entries); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, jOut.String())
	}
	if len(entries) == 0 {
		t.Error("expected at least one session in json list")
	}

	var eOut, eErr bytes.Buffer
	if code := runIncident([]string{"list"}, &eOut, &eErr); code != exitUserError {
		t.Errorf("list without --bundle: exit = %d, want %d", code, exitUserError)
	}
}

func TestRunIncidentReportShowsValidSignature(t *testing.T) {
	path := writeIncidentBundle(t)
	keyPath := writeSignTestPrivateKey(t, t.TempDir())
	var sOut, sErr bytes.Buffer
	if code := runSign([]string{"--bundle", path, "--key", keyPath}, &sOut, &sErr); code != exitSuccess {
		t.Fatalf("runSign exit = %d, stderr=%s", code, sErr.String())
	}

	var out, errBuf bytes.Buffer
	if code := runIncident([]string{"report", "--bundle", path, "--session", "sess-A"}, &out, &errBuf); code != exitSuccess {
		t.Fatalf("incident exit = %d, stderr=%s", code, errBuf.String())
	}
	got := strings.ToLower(out.String())
	if !strings.Contains(got, "signature: valid") {
		t.Errorf("expected a valid signature line in the report:\n%s", out.String())
	}
}

func TestRunIncidentReportErrors(t *testing.T) {
	path := writeIncidentBundle(t)
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"missing bundle", []string{"report", "--session", "sess-A"}, exitUserError},
		{"missing session", []string{"report", "--bundle", path}, exitUserError},
		{"bad format", []string{"report", "--bundle", path, "--session", "sess-A", "--format", "xml"}, exitUserError},
		{"session not found", []string{"report", "--bundle", path, "--session", "nope"}, exitUserError},
		{"unknown subcommand", []string{"frobnicate"}, exitUserError},
		{"no subcommand", []string{}, exitUserError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			if code := runIncident(tc.args, &out, &errBuf); code != tc.want {
				t.Errorf("exit = %d, want %d (stderr=%s)", code, tc.want, errBuf.String())
			}
		})
	}
}
