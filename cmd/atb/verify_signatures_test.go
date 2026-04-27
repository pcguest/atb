// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/signer"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

// signedBundleForVerify produces a bundle on disk signed via the local
// path so we can assert the verify-side rendering of provenance.
func signedBundleForVerify(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	b := buildCLIPrivilegedToolActionBundle(t)
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	if _, err := bundle.SignToWithSigner(context.Background(), path, path, signer.NewLocalSigner(priv)); err != nil {
		t.Fatalf("sign: %v", err)
	}
	return path
}

func TestRunVerify_LocalSignature_TextIncludesProvenance(t *testing.T) {
	path := signedBundleForVerify(t)

	var stdout, stderr bytes.Buffer
	exit := runVerify([]string{"--bundle", path}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("verify exit = %d, want success (stderr=%q)", exit, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "signature: backend=local") {
		t.Fatalf("expected text output to include 'signature: backend=local'; got:\n%s", out)
	}
	if !strings.Contains(out, "valid=true") {
		t.Fatalf("expected text output to include 'valid=true'; got:\n%s", out)
	}
}

func TestRunVerify_LocalSignature_JSONIncludesSignaturesArray(t *testing.T) {
	path := signedBundleForVerify(t)

	var stdout, stderr bytes.Buffer
	exit := runVerify([]string{"--bundle", path, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("verify exit = %d, want success (stderr=%q)", exit, stderr.String())
	}
	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if len(report.Signatures) != 1 {
		t.Fatalf("len(Signatures) = %d, want 1; payload:\n%s", len(report.Signatures), stdout.String())
	}
	sig := report.Signatures[0]
	if sig.Backend != "local" {
		t.Errorf("backend = %q, want local (default for legacy/local records)", sig.Backend)
	}
	if !sig.Valid {
		t.Errorf("valid = false, want true; error=%q", sig.Error)
	}
	if sig.PubKey == "" {
		t.Errorf("pubkey is empty")
	}
	// The local-signing path now records signed_at; assert it parses.
	if sig.SignedAt == "" {
		t.Errorf("signed_at empty; expected RFC 3339 timestamp from the local sign path")
	} else if _, err := time.Parse(time.RFC3339, sig.SignedAt); err != nil {
		t.Errorf("signed_at = %q, not RFC 3339: %v", sig.SignedAt, err)
	}
}

// TestRunVerify_RemoteSignature_TextProvenance constructs a synthetic
// signature record with backend / key_id / signed_at populated (as a
// remote signer would emit), without going through any actual remote
// signer, and confirms verify renders it correctly.
func TestRunVerify_RemoteSignature_TextProvenance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	b := buildCLIPrivilegedToolActionBundle(t)
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	// Re-load the saved bundle bytes, hash the prefix, and sign it as
	// any remote service would have done. This exercises the verify path
	// without standing up an HTTP server.
	rawBytes := mustReadFile(t, path)
	digest := sha256.Sum256(rawBytes)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sig := ed25519.Sign(priv, digest[:])

	loaded, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	signedAt := time.Date(2026, 4, 25, 10, 30, 0, 0, time.UTC).Format(time.RFC3339)
	record := map[string]any{
		"bundle_hash": hex.EncodeToString(digest[:]),
		"signature":   base64.StdEncoding.EncodeToString(sig),
		"pubkey":      base64.StdEncoding.EncodeToString(pub),
		"backend":     "https-http",
		"key_id":      "kms-prod-v3",
		"signed_at":   signedAt,
	}
	if err := loaded.AppendWithOptions(event.TypeBundleSignature, record, &bundle.AppendOptions{
		Timestamp: signedAt,
	}); err != nil {
		t.Fatalf("append signature record: %v", err)
	}
	if err := loaded.Save(path); err != nil {
		t.Fatalf("save after append: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runVerify([]string{"--bundle", path}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("verify exit = %d, want success (stderr=%q)\nstdout:\n%s", exit, stderr.String(), stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "backend=https-http") {
		t.Errorf("expected backend=https-http in text output:\n%s", out)
	}
	if !strings.Contains(out, "key_id=kms-prod-v3") {
		t.Errorf("expected key_id=kms-prod-v3 in text output:\n%s", out)
	}
	if !strings.Contains(out, "signed_at="+signedAt) {
		t.Errorf("expected signed_at=%s in text output:\n%s", signedAt, out)
	}
	if !strings.Contains(out, "valid=true") {
		t.Errorf("expected valid=true in text output:\n%s", out)
	}

	// Re-run with --json and assert the structured provenance.
	stdout.Reset()
	stderr.Reset()
	if exit := runVerify([]string{"--bundle", path, "--json"}, &stdout, &stderr); exit != exitSuccess {
		t.Fatalf("verify --json exit = %d, want success (stderr=%q)", exit, stderr.String())
	}
	var report verifypkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(report.Signatures) != 1 {
		t.Fatalf("len(Signatures) = %d, want 1", len(report.Signatures))
	}
	sigp := report.Signatures[0]
	if sigp.Backend != "https-http" || sigp.KeyID != "kms-prod-v3" || sigp.SignedAt != signedAt {
		t.Errorf("provenance mismatch: %+v", sigp)
	}
	if !sigp.Valid {
		t.Errorf("valid = false, want true; error = %q", sigp.Error)
	}
}

// mustReadFile is a tiny helper to keep the test bodies readable.
func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
