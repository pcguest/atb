// SPDX-License-Identifier: MIT
package main

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestRunCompliancePackWritesZip(t *testing.T) {
	b, err := bundle.New()
	if err != nil {
		t.Fatal(err)
	}
	add := func(eventType string, data map[string]any) {
		if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{
			Timestamp: "2026-06-15T00:00:00Z",
		}); err != nil {
			t.Fatalf("append %s: %v", eventType, err)
		}
	}
	add("ai.request.received", map[string]any{
		"request_id": "req-1", "actor_id_hash": "sha256:actor", "purpose_tag": "policy_decision",
	})
	add("ai.action.precommit", map[string]any{
		"action_id": "act-1", "action_type": "deploy",
		"action_parameters_digest": "sha256:params",
	})
	add("ai.policy.decision", map[string]any{
		"action_id": "act-1", "policy_id": "policy", "policy_version": "v1",
		"decision": "deny", "decision_reason_codes": []any{"review"},
		"subject_id_hash": "sha256:actor",
	})
	bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(bundlePath); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "pack.zip")
	var stdout, stderr bytes.Buffer
	code := runCompliance([]string{
		"pack",
		"--bundle", bundlePath,
		"--profile", "atb.profile.policy_decision",
		"--regime", "eu-ai-act",
		"--out", output,
	}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("runCompliance exit=%d stderr=%s", code, stderr.String())
	}
	reader, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	seen := map[string]bool{}
	for _, file := range reader.File {
		seen[file.Name] = true
	}
	for _, name := range []string{"bundle.atb", "MANIFEST.json", "reports/verify.report.json"} {
		if !seen[name] {
			t.Errorf("zip missing %q", name)
		}
	}
}
