// SPDX-License-Identifier: MIT
package incident_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/incident"
)

func TestBuildPack(t *testing.T) {
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	add := func(typ string, data map[string]any) {
		if err := b.AppendWithOptions(typ, data, &bundle.AppendOptions{Timestamp: "2026-05-28T09:00:00Z"}); err != nil {
			t.Fatalf("append %s: %v", typ, err)
		}
	}
	add(event.TypeCaptureScope, map[string]any{
		"targets": []any{"api.anthropic.com"}, "capture_mode": "digest",
		"out_of_scope": "Traffic that bypasses the proxy is not captured.",
	})
	add(event.TypeLLMRequest, map[string]any{"session_id": "sess-A", "host": "h", "method": "POST", "path": "/p"})
	add(event.TypeToolCall, map[string]any{"session_id": "sess-A", "tool_name": "wipe_db"})

	path := filepath.Join(t.TempDir(), "capture.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	files, err := incident.BuildPack(context.Background(), path, "sess-A", "9.9.9")
	if err != nil {
		t.Fatalf("BuildPack: %v", err)
	}

	byName := map[string][]byte{}
	for _, f := range files {
		byName[f.Name] = f.Content
	}
	for _, want := range []string{
		"bundle.atb", "incident-report.md", "incident-report.json",
		"verify-report.json", "README.md", "MANIFEST.json",
	} {
		if _, ok := byName[want]; !ok {
			t.Errorf("pack missing %q", want)
		}
	}

	var manifest incident.Manifest
	if err := json.Unmarshal(byName["MANIFEST.json"], &manifest); err != nil {
		t.Fatalf("manifest json: %v", err)
	}
	if manifest.ToolVersion != "9.9.9" || manifest.SessionID != "sess-A" {
		t.Errorf("manifest = %+v", manifest)
	}
	if manifest.ChainHeadHash == "" || !manifest.IntegrityValid {
		t.Errorf("manifest integrity fields: head=%q valid=%v", manifest.ChainHeadHash, manifest.IntegrityValid)
	}
	if manifest.CaptureScope == nil || manifest.CaptureScope.CaptureMode != "digest" {
		t.Errorf("manifest capture scope = %+v", manifest.CaptureScope)
	}
	// An unsigned bundle must say so plainly in the manifest, not omit the field
	// — the README claims a stated signature status, so it must be present.
	if manifest.SignatureStatus != "none (unsigned bundle)" {
		t.Errorf("manifest signature_status = %q, want unsigned statement", manifest.SignatureStatus)
	}

	// Every non-manifest file must be digested in the manifest, correctly.
	digests := map[string]string{}
	for _, fd := range manifest.Files {
		digests[fd.Name] = fd.SHA256
	}
	for name, content := range byName {
		if name == "MANIFEST.json" {
			continue
		}
		sum := sha256.Sum256(content)
		if digests[name] != hex.EncodeToString(sum[:]) {
			t.Errorf("manifest digest mismatch for %q", name)
		}
	}

	// The packed bundle must itself re-verify (independent evidence).
	packed := filepath.Join(t.TempDir(), "bundle.atb")
	if err := os.WriteFile(packed, byName["bundle.atb"], 0o644); err != nil {
		t.Fatalf("write packed bundle: %v", err)
	}
	loaded, err := bundle.Load(packed)
	if err != nil {
		t.Fatalf("load packed bundle: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Errorf("packed bundle failed independent verification: %v", err)
	}
}
