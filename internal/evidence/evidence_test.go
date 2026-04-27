// SPDX-License-Identifier: MIT
package evidence

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestBuildSimpleBundleEvidence(t *testing.T) {
	path := signedEvidenceBundle(t)

	ev, err := Build(context.Background(), path)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	wantPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("absolute path: %v", err)
	}
	if ev.Path != wantPath {
		t.Fatalf("Path = %q, want %q", ev.Path, wantPath)
	}
	if ev.Manifest.Version != 1 {
		t.Fatalf("manifest version = %d, want 1", ev.Manifest.Version)
	}
	if ev.Manifest.BundleID == "" {
		t.Fatal("manifest bundle_id is empty")
	}
	if ev.Manifest.CreatedAt == "" {
		t.Fatal("manifest created_at is empty")
	}
	if ev.RecordCount != 4 {
		t.Fatalf("RecordCount = %d, want 4", ev.RecordCount)
	}
	if ev.Tampered {
		t.Fatal("Tampered = true, want false")
	}
	if len(ev.Snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(ev.Snapshots))
	}
	if ev.Snapshots[0].Name != "checkpoint" {
		t.Fatalf("snapshot name = %q, want checkpoint", ev.Snapshots[0].Name)
	}
	if len(ev.Signatures) != 1 {
		t.Fatalf("signature count = %d, want 1", len(ev.Signatures))
	}
	if ev.Signatures[0].Backend != "local" {
		t.Fatalf("signature backend = %q, want local", ev.Signatures[0].Backend)
	}
	if !ev.Signatures[0].Valid {
		t.Fatalf("signature should be valid: %+v", ev.Signatures[0])
	}
}

func TestBuildMultiSignatureEvidence(t *testing.T) {
	path := signedEvidenceBundle(t)
	_, remotePrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate remote keypair: %v", err)
	}
	if _, err := bundle.SignToWithSigner(
		context.Background(),
		path,
		path,
		staticSigner{
			privateKey: remotePrivateKey,
			keyID:      "kms-key-7",
			backend:    "https-http",
		},
	); err != nil {
		t.Fatalf("remote sign bundle: %v", err)
	}

	ev, err := Build(context.Background(), path)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(ev.Signatures) != 2 {
		t.Fatalf("signature count = %d, want 2", len(ev.Signatures))
	}
	if ev.Signatures[0].Backend != "local" {
		t.Fatalf("first backend = %q, want local", ev.Signatures[0].Backend)
	}
	if ev.Signatures[1].Backend != "https-http" {
		t.Fatalf("second backend = %q, want https-http", ev.Signatures[1].Backend)
	}
	if ev.Signatures[1].KeyID != "kms-key-7" {
		t.Fatalf("second key_id = %q, want kms-key-7", ev.Signatures[1].KeyID)
	}
	if !ev.Signatures[0].Valid || !ev.Signatures[1].Valid {
		t.Fatalf("signatures should be valid: %+v", ev.Signatures)
	}
}

func TestBuildTamperedBundleEvidence(t *testing.T) {
	path := signedEvidenceBundle(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	tampered := bytes.Replace(raw, []byte("original"), []byte("changed"), 1)
	if bytes.Equal(tampered, raw) {
		t.Fatal("failed to tamper fixture")
	}
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatalf("write tampered bundle: %v", err)
	}

	ev, err := Build(context.Background(), path)
	if err == nil {
		t.Fatal("Build() error = nil, want tamper error")
	}
	if !ev.Tampered {
		t.Fatal("Tampered = false, want true")
	}
	if len(ev.Signatures) != 1 {
		t.Fatalf("signature count = %d, want 1", len(ev.Signatures))
	}
	if ev.Signatures[0].Valid {
		t.Fatalf("signature Valid = true, want false: %+v", ev.Signatures[0])
	}
}

func TestBuildMissingBundleReturnsZeroEvidence(t *testing.T) {
	ev, err := Build(context.Background(), filepath.Join(t.TempDir(), "missing.atb"))
	if err == nil {
		t.Fatal("Build() error = nil, want error")
	}
	if !reflect.DeepEqual(ev, BundleEvidence{}) {
		t.Fatalf("evidence = %+v, want zero value", ev)
	}
}

func signedEvidenceBundle(t testing.TB) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := b.AppendWithOptions("ai.tool.exec", map[string]any{
		"marker": "original",
	}, &bundle.AppendOptions{Timestamp: "2026-04-25T01:00:00Z"}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	snapshotHash, err := verifypkg.SnapshotBundleHash(b.Records)
	if err != nil {
		t.Fatalf("compute snapshot hash: %v", err)
	}
	if err := b.AppendWithOptions(event.TypeSnapshot, map[string]any{
		"name":         "checkpoint",
		"bundle_hash":  snapshotHash,
		"record_count": len(b.Records),
		"snapshot_at":  "2026-04-25T01:01:00Z",
	}, &bundle.AppendOptions{Timestamp: "2026-04-25T01:01:00Z"}); err != nil {
		t.Fatalf("append snapshot: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}
	if err := bundle.Sign(path, privateKey); err != nil {
		t.Fatalf("sign bundle: %v", err)
	}
	return path
}

type staticSigner struct {
	privateKey ed25519.PrivateKey
	keyID      string
	backend    string
}

func (s staticSigner) Sign(_ context.Context, digest []byte) ([]byte, []byte, string, string, string, error) {
	signature := ed25519.Sign(s.privateKey, digest)
	publicKey := s.privateKey.Public().(ed25519.PublicKey)
	return signature, publicKey, s.keyID, s.backend, "ed25519", nil
}
