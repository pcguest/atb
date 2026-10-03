// SPDX-License-Identifier: MIT

package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/event"
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

// TestManifestV3UsesStructuredDataAndVerifies pins the acquisition-aware
// canonical profile declaration: v3 uses the structured wire form, round-trips,
// and verifies.
func TestManifestV3UsesStructuredDataAndVerifies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := NewWithOptions(NewOptions{ManifestVersion: ManifestVersionV3})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	data, ok := b.Records[0].Event.Data.(map[string]any)
	if !ok {
		t.Fatalf("manifest v3 data type = %T, want map[string]any", b.Records[0].Event.Data)
	}
	if v, _ := data["version"].(int); v != ManifestVersionV3 {
		t.Fatalf("manifest version = %v, want %d", data["version"], ManifestVersionV3)
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
	if manifest := loaded.Manifest(); manifest == nil || manifest.Version != "3" {
		t.Fatalf("manifest = %+v, want version 3", manifest)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

// TestManifestVersionMaxIsV3 pins the forward-compatibility guard: v3 is
// accepted, and a version above the supported maximum is rejected with
// ErrMalformed so a stale reader fails loudly instead of silently mis-hashing.
func TestManifestVersionMaxIsV3(t *testing.T) {
	if ManifestVersionMax != ManifestVersionV3 {
		t.Fatalf("ManifestVersionMax = %d, want %d", ManifestVersionMax, ManifestVersionV3)
	}
	b, err := NewWithOptions(NewOptions{ManifestVersion: ManifestVersionV3})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	if _, err := parseManifestData(&b.Records[0]); err != nil {
		t.Fatalf("v3 manifest rejected: %v", err)
	}
	data := b.Records[0].Event.Data.(map[string]any)
	data["version"] = ManifestVersionV3 + 1
	if _, err := parseManifestData(&b.Records[0]); !errors.Is(err, ErrMalformed) {
		t.Fatalf("forward-bumped manifest: want ErrMalformed, got %v", err)
	}
}

// TestAppendAcquisitionToPreV3BundleRefused pins the bundle-layer version floor:
// appending an acquisition-bearing event to a pre-v3 bundle is refused, so no
// caller can build a mixed-version chain.
func TestAppendAcquisitionToPreV3BundleRefused(t *testing.T) {
	b, err := New() // v1
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = b.AppendWithOptions("ai.tool.exec", map[string]any{"ok": true}, &AppendOptions{
		Acquisition: &event.AcquisitionInfo{Mode: "retrospective"},
	})
	if err == nil || !strings.Contains(err.Error(), "manifest version < 3") {
		t.Fatalf("want manifest-version-floor error, got %v", err)
	}
}

// TestAppendAcquisitionToV3BundleAllowed pins the positive case.
func TestAppendAcquisitionToV3BundleAllowed(t *testing.T) {
	b, err := NewWithOptions(NewOptions{ManifestVersion: ManifestVersionV3})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	if err := b.AppendWithOptions("ai.tool.exec", map[string]any{"ok": true}, &AppendOptions{
		Acquisition: &event.AcquisitionInfo{Mode: "retrospective", SourceSystem: "chatlog"},
	}); err != nil {
		t.Fatalf("v3 append acquisition: %v", err)
	}
	if err := b.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

// manifestRecordFromStringData builds a manifest Record from a JSON-encoded
// data payload (used to test string-form rejection/acceptance without going
// through the writer).
func manifestRecordFromStringData(t *testing.T, data string) Record {
	t.Helper()
	line := `{"event":{"seq":0,"prev_hash":"0000000000000000000000000000000000000000000000000000000000000000","type":"atb.bundle.manifest","hash_algo":"sha256","data":` + jsonEscape(data) + `},"hash":"0000000000000000000000000000000000000000000000000000000000000000"}`
	var rec Record
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("unmarshal manifest record: %v", err)
	}
	return rec
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestManifestStringEncodedV2V3Rejected pins the string-form hardening: a
// manifest whose extracted version is 2 or 3 must be a structured JSON object.
// If it is presented as a JSON string, parseManifestData rejects it with
// ErrMalformed rather than silently falling back to the v1 parser.
func TestManifestStringEncodedV2V3Rejected(t *testing.T) {
	cases := []int{ManifestVersionV2, ManifestVersionV3}
	for _, v := range cases {
		t.Run(fmt.Sprintf("v%d", v), func(t *testing.T) {
			var payload string
			if v == ManifestVersionV2 {
				payload = `{"version":2,"created_at":"2026-04-26T00:00:00Z","bundle_id":"00112233445566778899aabbccddeeff"}`
			} else {
				payload = `{"version":3,"created_at":"2026-04-26T00:00:00Z","bundle_id":"00112233445566778899aabbccddeeff"}`
			}
			rec := manifestRecordFromStringData(t, payload)
			_, err := parseManifestData(&rec)
			if err == nil {
				t.Fatal("expected string-encoded structured manifest to be rejected")
			}
			if !errors.Is(err, ErrMalformed) {
				t.Fatalf("want ErrMalformed, got %v", err)
			}
			if !strings.Contains(err.Error(), "structured JSON object") {
				t.Fatalf("expected structured-object error, got %v", err)
			}
		})
	}
}

// TestManifestStringEncodedV1StillAccepted preserves genuine legacy v1 string
// manifests: a JSON string payload with version "1" (or no version field)
// continues to parse.
func TestManifestStringEncodedV1StillAccepted(t *testing.T) {
	rec := manifestRecordFromStringData(t, `{"version":"1","created_at":"2026-04-26T00:00:00Z","bundle_id":"00112233445566778899aabbccddeeff"}`)
	manifest, err := parseManifestData(&rec)
	if err != nil {
		t.Fatalf("v1 string manifest rejected: %v", err)
	}
	if manifest.Version != "1" {
		t.Fatalf("version = %q, want 1", manifest.Version)
	}
	if manifest.BundleID != "00112233445566778899aabbccddeeff" {
		t.Fatalf("bundle_id = %q", manifest.BundleID)
	}
}

// TestAppendAcquisitionToManifestLessBundleRefused pins the floor for legacy
// manifest-less bundles: they have no declared profile, so an acquisition-
// bearing append must be refused, while normal (non-acquisition) appends keep
// working.
func TestAppendAcquisitionToManifestLessBundleRefused(t *testing.T) {
	b := &Bundle{}
	if err := b.Append("ai.tool.exec", map[string]any{"ok": true}); err != nil {
		t.Fatalf("non-acquisition append to manifest-less bundle failed: %v", err)
	}
	err := b.AppendWithOptions("ai.tool.exec", map[string]any{"ok": true}, &AppendOptions{
		Acquisition: &event.AcquisitionInfo{Mode: "retrospective"},
	})
	if err == nil || !strings.Contains(err.Error(), "no manifest record") {
		t.Fatalf("want no-manifest-record refusal, got %v", err)
	}
	if len(b.Records) != 1 {
		t.Fatalf("refused append mutated bundle: got %d records", len(b.Records))
	}
}
