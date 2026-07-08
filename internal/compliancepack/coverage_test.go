// SPDX-License-Identifier: MIT
package compliancepack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/retentionaudit"
)

func TestBuildAndProfileValidationContracts(t *testing.T) {
	tests := []struct {
		name       string
		bundlePath string
		profile    string
		regime     string
		want       string
	}{
		{name: "bundle required", profile: "policy_decision", want: "bundle path is required"},
		{name: "profile required", bundlePath: "bundle.atb", want: "profile is required"},
		{name: "regime", bundlePath: "bundle.atb", profile: "policy_decision", regime: "nist", want: "unsupported regime"},
		{name: "missing bundle", bundlePath: "missing.atb", profile: "policy_decision", want: "load bundle"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			//lint:ignore SA1012 Build explicitly accepts nil for backward-compatible callers.
			_, err := Build(nil, tc.bundlePath, tc.profile, tc.regime, "test")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%q", err, tc.want)
			}
		})
	}

	profile, err := resolveProfile("policy_decision")
	if err != nil || profile.ID() != "atb.profile.policy_decision" {
		t.Fatalf("short profile=%v err=%v", profile, err)
	}
	profile, err = resolveProfile(filepath.Join("..", "profiles", "templates", "rag_answer.yaml"))
	if err != nil || profile == nil {
		t.Fatalf("file profile=%v err=%v", profile, err)
	}
	if _, err := resolveProfile("does-not-exist"); err == nil || !strings.Contains(err.Error(), "resolve profile") {
		t.Fatalf("unknown profile error=%v", err)
	}
}

func TestRetentionArtifactAndFormattingBoundaries(t *testing.T) {
	dir := t.TempDir()
	evidencePath := filepath.Join(dir, bundle.BundleDir, bundle.BundleFile)
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0o750); err != nil {
		t.Fatal(err)
	}
	auditPath := retentionaudit.PathForBundle(evidencePath)
	if err := os.MkdirAll(filepath.Dir(auditPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(auditPath, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := appendRetentionFiles(nil, evidencePath, "hash"); err == nil ||
		!strings.Contains(err.Error(), "load retention audit") {
		t.Fatalf("corrupt audit error=%v", err)
	}

	if err := os.Remove(auditPath); err != nil {
		t.Fatal(err)
	}
	audit, err := bundle.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := audit.AppendWithOptions("custom.event", map[string]any{"ok": true}, &bundle.AppendOptions{
		Timestamp: "2026-06-30T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := audit.Save(context.Background(), auditPath); err != nil {
		t.Fatal(err)
	}
	files, err := appendRetentionFiles([]File{{Name: "existing"}}, evidencePath, "hash")
	if err != nil || len(files) != 1 {
		t.Fatalf("irrelevant retention files=%v err=%v", files, err)
	}

	if safeName("") != "session" || safeName("session/with spaces") != "session_with_spaces" {
		t.Fatal("safeName normalization failed")
	}
	if passLabel(true) != "PASS" || passLabel(false) != "FAIL" {
		t.Fatal("pass labels failed")
	}
	if _, err := marshalJSON(func() {}); err == nil {
		t.Fatal("unmarshalable compliance JSON succeeded")
	}

	if got := bundleTimestamp(nil); !got.Equal(time.Unix(0, 0).UTC()) {
		t.Fatalf("nil timestamp=%v", got)
	}
	b := &bundle.Bundle{Records: []bundle.Record{{}}}
	if got := bundleTimestamp(b); !got.Equal(time.Unix(0, 0).UTC()) {
		t.Fatalf("invalid timestamp=%v", got)
	}
	b.Records[0].Event.Timestamp = "2026-06-30T01:02:03.123Z"
	if got := bundleTimestamp(b); got.Format(time.RFC3339Nano) != "2026-06-30T01:02:03.123Z" {
		t.Fatalf("valid timestamp=%v", got)
	}
}
