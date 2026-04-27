// SPDX-License-Identifier: MIT
package sign

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestVerifyBundleSignature_Ed25519(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	digest := sha256.Sum256([]byte("ed25519 bundle bytes"))
	sig := ed25519.Sign(priv, digest[:])

	bs := BundleSignature{
		BundleHash: hex.EncodeToString(digest[:]),
		Signature:  base64.StdEncoding.EncodeToString(sig),
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
		Algorithm:  "ed25519",
	}
	if err := VerifyBundleSignature(bs, digest[:]); err != nil {
		t.Fatalf("ed25519 verify: %v", err)
	}

	// Missing algorithm field defaults to ed25519.
	bs.Algorithm = ""
	if err := VerifyBundleSignature(bs, digest[:]); err != nil {
		t.Fatalf("default algorithm verify: %v", err)
	}
}

func TestVerifyBundleSignature_ECDSAP256_Uncompressed(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	digest := sha256.Sum256([]byte("ecdsa-p256 bundle bytes"))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	//lint:ignore SA1019 SEC1 uncompressed P-256 is the wire format the verifier accepts; crypto/ecdh's ECDH-only API cannot construct this byte sequence from an ecdsa.PrivateKey
	rawPub := elliptic.Marshal(elliptic.P256(), priv.X, priv.Y)

	bs := BundleSignature{
		BundleHash: hex.EncodeToString(digest[:]),
		Signature:  base64.StdEncoding.EncodeToString(sig),
		PublicKey:  base64.StdEncoding.EncodeToString(rawPub),
		Algorithm:  "ecdsa-p256",
	}
	if err := VerifyBundleSignature(bs, digest[:]); err != nil {
		t.Fatalf("ecdsa uncompressed verify: %v", err)
	}
}

func TestVerifyBundleSignature_ECDSAP256_PKIX(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	digest := sha256.Sum256([]byte("pkix payload"))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	pkix, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}

	bs := BundleSignature{
		BundleHash: hex.EncodeToString(digest[:]),
		Signature:  base64.StdEncoding.EncodeToString(sig),
		PublicKey:  base64.StdEncoding.EncodeToString(pkix),
		Algorithm:  "ecdsa-p256",
	}
	if err := VerifyBundleSignature(bs, digest[:]); err != nil {
		t.Fatalf("ecdsa pkix verify: %v", err)
	}
}

func TestVerifyBundleSignature_ECDSAP256_Tampered(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	original := []byte("original bundle bytes")
	digest := sha256.Sum256(original)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	//lint:ignore SA1019 SEC1 uncompressed P-256 is the wire format the verifier accepts; crypto/ecdh's ECDH-only API cannot construct this byte sequence from an ecdsa.PrivateKey
	rawPub := elliptic.Marshal(elliptic.P256(), priv.X, priv.Y)

	// Flip a bit by re-hashing different content but keep stored hash
	// matching the tampered content (so we reach signature verify and it fails).
	tampered := append([]byte{}, original...)
	tampered[0] ^= 0x01
	tamperedDigest := sha256.Sum256(tampered)

	bs := BundleSignature{
		BundleHash: hex.EncodeToString(tamperedDigest[:]),
		Signature:  base64.StdEncoding.EncodeToString(sig),
		PublicKey:  base64.StdEncoding.EncodeToString(rawPub),
		Algorithm:  "ecdsa-p256",
	}
	err = VerifyBundleSignature(bs, tamperedDigest[:])
	if err == nil {
		t.Fatal("expected verification to fail on tampered payload")
	}
	if !errors.Is(err, ErrTamper) {
		t.Fatalf("expected ErrTamper, got %v", err)
	}
}

func TestVerifyBundleSignature_UnknownAlgorithm(t *testing.T) {
	digest := sha256.Sum256([]byte("x"))
	bs := BundleSignature{
		BundleHash: hex.EncodeToString(digest[:]),
		Signature:  base64.StdEncoding.EncodeToString([]byte("dummy")),
		PublicKey:  base64.StdEncoding.EncodeToString([]byte("dummy")),
		Algorithm:  "rsa-pss-sha512",
	}
	err := VerifyBundleSignature(bs, digest[:])
	if err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
	if errors.Is(err, ErrTamper) {
		t.Fatalf("unsupported algorithm must not be wrapped in ErrTamper: %v", err)
	}
	if !strings.Contains(err.Error(), "unsupported algorithm") {
		t.Fatalf("expected 'unsupported algorithm' in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "upgrade atb") {
		t.Fatalf("expected actionable upgrade hint, got %v", err)
	}
}

func TestNewBundleSignatureRecord_LocalDefaults(t *testing.T) {
	digest := sha256.Sum256([]byte("local"))
	pub := make([]byte, ed25519.PublicKeySize)
	sig := make([]byte, ed25519.SignatureSize)

	rec := NewBundleSignatureRecord(digest[:], pub, sig, "", "", "", "")

	if got := rec["algorithm"]; got != "ed25519" {
		t.Errorf("algorithm = %q, want %q", got, "ed25519")
	}
	if got := rec["backend"]; got != "local" {
		t.Errorf("backend = %q, want %q", got, "local")
	}
	if rec["signed_at"] == "" {
		t.Errorf("signed_at must be set on local records, got empty")
	}
	if _, err := time.Parse(time.RFC3339Nano, rec["signed_at"]); err != nil {
		t.Errorf("signed_at not RFC3339Nano: %q (%v)", rec["signed_at"], err)
	}
	if _, present := rec["key_id"]; present {
		t.Errorf("key_id must be omitted for local signatures, got %q", rec["key_id"])
	}
}

func TestNewBundleSignatureRecord_KMSProvenance(t *testing.T) {
	digest := sha256.Sum256([]byte("kms"))
	pub := make([]byte, 65)
	pub[0] = 0x04
	sig := make([]byte, 70)

	rec := NewBundleSignatureRecord(
		digest[:], pub, sig,
		"arn:aws:kms:us-east-1:111122223333:key/abcd-1234",
		"aws-kms",
		"ecdsa-p256",
		"2026-04-26T12:00:00.123456789Z",
	)
	if rec["algorithm"] != "ecdsa-p256" {
		t.Errorf("algorithm = %q", rec["algorithm"])
	}
	if rec["backend"] != "aws-kms" {
		t.Errorf("backend = %q", rec["backend"])
	}
	if !strings.HasPrefix(rec["key_id"], "arn:aws:kms:") {
		t.Errorf("key_id = %q", rec["key_id"])
	}
	if rec["signed_at"] != "2026-04-26T12:00:00.123456789Z" {
		t.Errorf("signed_at = %q", rec["signed_at"])
	}
}

func TestParseBundleSignature_MissingAlgorithmDefaultsToEd25519(t *testing.T) {
	bs, err := ParseBundleSignature(map[string]any{
		"bundle_hash": strings.Repeat("0", 64),
		"signature":   base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
		"pubkey":      base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize)),
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if bs.Algorithm != "ed25519" {
		t.Errorf("Algorithm = %q, want ed25519", bs.Algorithm)
	}
	if bs.Backend != "" || bs.KeyID != "" || bs.SignedAt != "" {
		t.Errorf("expected new fields empty for legacy record, got Backend=%q KeyID=%q SignedAt=%q",
			bs.Backend, bs.KeyID, bs.SignedAt)
	}
}

func TestParseBundleSignature_PopulatesProvenanceFields(t *testing.T) {
	bs, err := ParseBundleSignature(map[string]any{
		"bundle_hash": strings.Repeat("0", 64),
		"signature":   base64.StdEncoding.EncodeToString(make([]byte, 70)),
		"pubkey":      base64.StdEncoding.EncodeToString(make([]byte, 65)),
		"algorithm":   "ecdsa-p256",
		"backend":     "vault",
		"key_id":      "transit/keys/atb-prod",
		"signed_at":   "2026-04-26T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if bs.Algorithm != "ecdsa-p256" {
		t.Errorf("Algorithm = %q", bs.Algorithm)
	}
	if bs.Backend != "vault" {
		t.Errorf("Backend = %q", bs.Backend)
	}
	if bs.KeyID != "transit/keys/atb-prod" {
		t.Errorf("KeyID = %q", bs.KeyID)
	}
	if bs.SignedAt != "2026-04-26T12:00:00Z" {
		t.Errorf("SignedAt = %q", bs.SignedAt)
	}
}

func TestParseBundleSignature_TolerantOfUnknownFields(t *testing.T) {
	bs, err := ParseBundleSignature(map[string]any{
		"bundle_hash":     strings.Repeat("0", 64),
		"signature":       base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
		"pubkey":          base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize)),
		"future_field":    "ignored",
		"another_unknown": 12345,
		"nested":          map[string]any{"x": "y"},
	})
	if err != nil {
		t.Fatalf("expected lenient parse, got error: %v", err)
	}
	if bs.BundleHash == "" {
		t.Error("expected mandatory fields to still populate")
	}
}
