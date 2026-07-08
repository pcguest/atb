// SPDX-License-Identifier: MIT
package compliancepack

import (
	"bytes"
	"testing"
	"time"
)

func TestAddArtifactRegeneratesManifestAndChecksumsImmutably(t *testing.T) {
	originalContent := []byte("bundle")
	original := Pack{
		GeneratedAt: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
		Manifest: Manifest{
			PackVersion: "compliance.pack.v1",
			BundleFile:  "bundle.atb",
		},
		Files: []File{
			{Name: "bundle.atb", Content: originalContent},
			{Name: "MANIFEST.json", Content: []byte("old manifest")},
			{Name: "SHA256SUMS", Content: []byte("old sums")},
		},
	}
	receipt := []byte(`{"receipt_id":"r1"}`)
	got, err := AddArtifact(original, File{Name: "mortise/receipt.json", Content: receipt})
	if err != nil {
		t.Fatal(err)
	}
	originalContent[0] = 'X'
	receipt[0] = 'X'

	names := map[string][]byte{}
	for _, file := range got.Files {
		names[file.Name] = file.Content
	}
	for _, name := range []string{"bundle.atb", "mortise/receipt.json", "MANIFEST.json", "SHA256SUMS"} {
		if _, ok := names[name]; !ok {
			t.Errorf("missing %s", name)
		}
	}
	if string(names["bundle.atb"]) != "bundle" || !bytes.Contains(names["mortise/receipt.json"], []byte(`"receipt_id"`)) {
		t.Fatal("AddArtifact retained mutable input aliases")
	}
	if !bytes.Contains(names["MANIFEST.json"], []byte(`"mortise/receipt.json"`)) ||
		!bytes.Contains(names["SHA256SUMS"], []byte("mortise/receipt.json")) {
		t.Fatal("receipt is not covered by manifest and checksums")
	}
	if len(original.Files) != 3 || string(original.Files[1].Content) != "old manifest" {
		t.Fatal("original pack was mutated")
	}
}

func TestAddArtifactRejectsReservedAndDuplicateNames(t *testing.T) {
	pack := Pack{Files: []File{{Name: "bundle.atb", Content: []byte("bundle")}}}
	for _, name := range []string{"", "MANIFEST.json", "SHA256SUMS", "bundle.atb"} {
		if _, err := AddArtifact(pack, File{Name: name}); err == nil {
			t.Errorf("AddArtifact accepted %q", name)
		}
	}
}
