// SPDX-License-Identifier: MIT
package receipt

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func newTestReceipt() Receipt {
	return Receipt{
		ReceiptID:   "sha256-abc",
		BundleHash:  "abc123",
		SubmittedAt: time.Date(2026, time.May, 28, 9, 0, 0, 0, time.UTC),
	}
}

func TestAttestAndVerifyRoundTrip(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := NewSigner(priv)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	r := newTestReceipt()
	att := signer.Attest(r, time.Now())
	r.Attestation = &att

	if att.Algorithm != "ed25519" || att.PubKey == "" || att.Signature == "" || att.SignedAt == "" {
		t.Fatalf("incomplete attestation: %+v", att)
	}
	if err := VerifyAttestation(r); err != nil {
		t.Errorf("VerifyAttestation: %v", err)
	}
}

func TestVerifyAttestationDetectsTamper(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := NewSigner(priv)
	r := newTestReceipt()
	att := signer.Attest(r, time.Now())
	r.Attestation = &att

	// Mutate a custody fact after attestation.
	r.BundleHash = "deadbeef"
	if err := VerifyAttestation(r); err == nil {
		t.Error("expected verification failure after tampering with bundle_hash")
	}
}

func TestVerifyAttestationMissing(t *testing.T) {
	if err := VerifyAttestation(newTestReceipt()); err == nil {
		t.Error("expected error for a receipt with no attestation")
	}
}

func TestNewSignerRejectsBadKey(t *testing.T) {
	if _, err := NewSigner(make(ed25519.PrivateKey, 10)); err == nil {
		t.Error("expected error for an invalid private key size")
	}
}
