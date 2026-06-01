// SPDX-License-Identifier: MIT
package receipt

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// Attestation is Custos's Ed25519 signature over a receipt's core custody
// facts. It is the independent proof that Custos received bundle BundleHash at
// SubmittedAt — verifiable by any holder against the embedded public key,
// without trusting the receipt store.
type Attestation struct {
	Algorithm string `json:"algorithm"` // "ed25519"
	PubKey    string `json:"pubkey"`    // base64-std Ed25519 public key
	SignedAt  string `json:"signed_at"` // RFC 3339
	Signature string `json:"signature"` // base64-std signature over the custody message
}

// Signer issues custody attestations with an Ed25519 private key.
type Signer struct {
	priv ed25519.PrivateKey
}

// NewSigner returns a Signer for a valid Ed25519 private key.
func NewSigner(priv ed25519.PrivateKey) (*Signer, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, errors.New("custos: invalid ed25519 private key size")
	}
	return &Signer{priv: priv}, nil
}

// attestationMessage is the canonical byte string Custos signs. Changing it is
// a breaking change to attestation verification.
func attestationMessage(bundleHash, receiptID, submittedAt, signedAt string) []byte {
	return []byte(bundleHash + "\n" + receiptID + "\n" + submittedAt + "\n" + signedAt)
}

// Attest returns a signed attestation over the receipt's custody facts.
func (s *Signer) Attest(r Receipt, signedAt time.Time) Attestation {
	at := signedAt.UTC().Format(time.RFC3339)
	msg := attestationMessage(r.BundleHash, r.ReceiptID, r.SubmittedAt, at)
	sig := ed25519.Sign(s.priv, msg)
	pub := s.priv.Public().(ed25519.PublicKey)
	return Attestation{
		Algorithm: "ed25519",
		PubKey:    base64.StdEncoding.EncodeToString(pub),
		SignedAt:  at,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}
}

// VerifyAttestation checks a receipt's attestation against its custody facts and
// the public key embedded in the attestation.
func VerifyAttestation(r Receipt) error {
	a := r.Attestation
	if a == nil {
		return errors.New("custos: receipt has no attestation")
	}
	if a.Algorithm != "ed25519" {
		return fmt.Errorf("custos: unsupported attestation algorithm %q", a.Algorithm)
	}
	pub, err := base64.StdEncoding.DecodeString(a.PubKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("custos: invalid attestation public key")
	}
	sig, err := base64.StdEncoding.DecodeString(a.Signature)
	if err != nil {
		return errors.New("custos: invalid attestation signature encoding")
	}
	msg := attestationMessage(r.BundleHash, r.ReceiptID, r.SubmittedAt, a.SignedAt)
	if !ed25519.Verify(ed25519.PublicKey(pub), msg, sig) {
		return errors.New("custos: attestation signature does not verify")
	}
	return nil
}
