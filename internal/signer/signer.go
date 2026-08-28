// SPDX-License-Identifier: MIT

// Package signer defines the abstraction every ATB bundle-signing backend
// must satisfy. The interface is the seam point for plugging in remote key
// management backends (AWS KMS, GCP Cloud KMS, Vault, Azure Key Vault) while
// keeping local Ed25519 signing as the default and primary path.
//
// Contract (also enforced by docs/specification/bundle-v1.md §4.2):
//
//   - The bytes passed to Sign are the canonical pre-image as defined by the
//     bundle layer. Today that pre-image is the 32-byte SHA-256 digest of the
//     pre-signature bundle NDJSON; cmd/atb/sign passes that digest in via
//     bundle.SignToWithSigner. Implementations MUST sign the bytes they are
//     handed verbatim — they MUST NOT re-hash, salt, or wrap them.
//   - Ed25519 in Go's crypto/ed25519 signs the message argument verbatim. We
//     pass it the 32-byte digest, so the on-the-wire signature is over the
//     digest, not over the raw bundle. This matches the legacy behaviour of
//     internal/bundle/sign.go on main.
//   - Implementations MUST return the verification public key in raw form
//     (32 bytes for Ed25519) so verifiers can validate without a live call
//     to the backend.
package signer

import (
	"context"
	"crypto/ed25519"
	"errors"
)

// ErrInvalidKey is returned when a Signer is constructed or invoked with a
// key that is not the expected size or material.
var ErrInvalidKey = errors.New("signer: invalid key material")

// Signer is the single interface all signing backends must satisfy.
type Signer interface {
	// Sign signs digest and returns:
	//   sig     — the raw signature bytes (64 bytes for Ed25519).
	//   pubKey  — the raw public key bytes (32 bytes for Ed25519).
	//   keyID   — opaque backend-scoped key identifier; empty for local.
	//   backend — short backend name. Empty string is the legacy/implicit
	//             local default and is omitted from the on-disk signature
	//             record so that bundles signed by LocalSigner remain
	//             byte-identical to bundles signed before this interface
	//             existed. KMS-style backends MUST return a non-empty
	//             backend identifier (e.g., "aws-kms").
	//   algorithm — signing algorithm, e.g. "ed25519" or "ecdsa-p256".
	Sign(ctx context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error)
}

// LocalSigner implements Signer using a local Ed25519 private key. This is
// the default and primary signing backend.
type LocalSigner struct {
	privateKey ed25519.PrivateKey
}

// NewLocalSigner returns a LocalSigner backed by the given Ed25519 private
// key. It does not validate the key here; Sign returns ErrInvalidKey if the
// key is missing or the wrong size.
func NewLocalSigner(privateKey ed25519.PrivateKey) *LocalSigner {
	return &LocalSigner{privateKey: privateKey}
}

// Sign implements Signer for the local Ed25519 backend.
//
// The returned backend is the empty string deliberately: the on-disk bundle
// signature record on main does not carry a backend field, and emitting
// backend="local" for every newly-signed bundle would change the on-disk
// format for the local path. Empty-string is the documented "legacy/implicit
// local" sentinel; KMS signers will return their explicit backend name.
//
// A future manifest migration may replace the implicit local sentinel with an
// explicit backend="local" field. See docs/specification/bundle-v1.md §4.2.
func (s *LocalSigner) Sign(_ context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error) {
	if len(s.privateKey) != ed25519.PrivateKeySize {
		return nil, nil, "", "", "", ErrInvalidKey
	}
	// Ed25519 in Go signs the message argument verbatim. The bundle layer
	// passes the 32-byte SHA-256 digest as the message — see
	// internal/bundle/sign.go.
	sig = ed25519.Sign(s.privateKey, digest)
	pub, ok := s.privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, nil, "", "", "", ErrInvalidKey
	}
	pubKey = append([]byte(nil), pub...) // defensive copy
	return sig, pubKey, "", "", "ed25519", nil
}
