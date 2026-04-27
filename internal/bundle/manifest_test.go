// SPDX-License-Identifier: MIT

package bundle

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestManifestV2UsesStructuredData(t *testing.T) {
	b, err := NewWithOptions(NewOptions{ManifestVersion: ManifestVersionV2})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	data, ok := b.Records[0].Event.Data.(map[string]any)
	if !ok {
		t.Fatalf("manifest data type = %T, want map[string]any", b.Records[0].Event.Data)
	}
	if _, ok := data["version"].(int); !ok {
		t.Fatalf("manifest version type = %T, want int", data["version"])
	}
	if _, ok := b.Records[0].Event.Data.(string); ok {
		t.Fatalf("manifest v2 data must not be a JSON-encoded string")
	}
}

func TestManifestV2LoadAndVerify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := NewWithOptions(NewOptions{ManifestVersion: ManifestVersionV2})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"ok": true}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	manifest := loaded.Manifest()
	if manifest == nil {
		t.Fatal("expected manifest")
	}
	if manifest.Version != "2" {
		t.Fatalf("manifest version = %q, want 2", manifest.Version)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestManifestV1LoadRegression(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := b.Records[0].Event.Data.(string); !ok {
		t.Fatalf("manifest v1 data type = %T, want string", b.Records[0].Event.Data)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if manifest := loaded.Manifest(); manifest == nil || manifest.Version != ManifestVersion {
		t.Fatalf("manifest = %+v, want version %s", manifest, ManifestVersion)
	}
}

func TestManifestV2RejectedByV1Path(t *testing.T) {
	raw, err := json.Marshal(ManifestData{
		Version:   "2",
		CreatedAt: "2026-04-26T00:00:00Z",
		BundleID:  "00112233445566778899aabbccddeeff",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if _, err := parseManifestDataV1(string(raw)); err == nil {
		t.Fatal("expected v1-only parser to reject version 2")
	}
}
