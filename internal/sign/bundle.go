// SPDX-License-Identifier: MIT
package sign

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrTamper indicates that a bundle's signature record does not validate
// against the bundle bytes — i.e. content has been altered after signing.
// Capability gaps (e.g. unsupported algorithms) are NOT wrapped in this
// error; only true tamper signals are.
var ErrTamper = errors.New("bundle signature: tamper detected")

// BundleSignature is the parsed in-memory view of an atb.bundle.signature
// data payload. JSON tags reflect the on-disk wire format. New optional
// provenance fields (Algorithm, KeyID, Backend, SignedAt) are emitted with
// `omitempty`; they default to the empty string for legacy bundles, in
// which case Algorithm is treated as "ed25519" by the verifier.
type BundleSignature struct {
	BundleHash string `json:"bundle_hash"`
	Signature  string `json:"signature"`
	PublicKey  string `json:"pubkey"`
	Algorithm  string `json:"algorithm,omitempty"`
	KeyID      string `json:"key_id,omitempty"`
	Backend    string `json:"backend,omitempty"`
	SignedAt   string `json:"signed_at,omitempty"`
}

// NewBundleSignatureRecord builds the data payload for an atb.bundle.signature
// record. The provenance fields (algorithm, backend, signedAt, keyID) are
// documented in docs/specification/bundle-v1.md §4.2.
//
// Defaults applied when the caller passes empty strings:
//   - algorithm → "ed25519"
//   - backend   → "local"
//   - signedAt  → time.Now().UTC().Format(time.RFC3339Nano)
//
// keyID is emitted only when non-empty (local Ed25519 signatures have no
// backend-scoped identifier). algorithm and backend are always emitted on
// new bundles so verifiers can attribute the signature without guessing.
// Existing bundles signed before this change omit these fields and remain
// fully verifiable — the reader treats absent algorithm as "ed25519".
func NewBundleSignatureRecord(bundleHash []byte, publicKey []byte, signature []byte, keyID, backend, algorithm, signedAt string) map[string]string {
	if algorithm == "" {
		algorithm = "ed25519"
	}
	if backend == "" {
		backend = "local"
	}
	if signedAt == "" {
		signedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	rec := map[string]string{
		"bundle_hash": hex.EncodeToString(bundleHash),
		"signature":   base64.StdEncoding.EncodeToString(signature),
		"pubkey":      base64.StdEncoding.EncodeToString(publicKey),
		"algorithm":   algorithm,
		"backend":     backend,
		"signed_at":   signedAt,
	}
	if keyID != "" {
		rec["key_id"] = keyID
	}
	return rec
}

func ParseBundleSignature(data interface{}) (BundleSignature, error) {
	fields, ok := data.(map[string]any)
	if !ok {
		return BundleSignature{}, fmt.Errorf("bundle signature data is not an object")
	}

	bundleHash := bundleSignatureField(fields, "bundle_hash")
	if bundleHash == "" {
		return BundleSignature{}, fmt.Errorf("bundle_hash absent")
	}

	signature := bundleSignatureField(fields, "signature")
	if signature == "" {
		return BundleSignature{}, fmt.Errorf("signature absent")
	}

	publicKey := bundleSignatureField(fields, "pubkey")
	if publicKey == "" {
		// Backward compatibility for bundles signed before the CLI wrote `pubkey`.
		publicKey = bundleSignatureField(fields, "public_key")
	}
	if publicKey == "" {
		return BundleSignature{}, fmt.Errorf("pubkey absent")
	}

	return BundleSignature{
		BundleHash: bundleHash,
		Signature:  signature,
		PublicKey:  publicKey,
		Algorithm:  bundleSignatureAlgorithm(fields),
		KeyID:      bundleSignatureField(fields, "key_id"),
		Backend:    bundleSignatureField(fields, "backend"),
		SignedAt:   bundleSignatureField(fields, "signed_at"),
	}, nil
}

func VerifyBundleSignature(signature BundleSignature, expectedBundleHash []byte) error {
	if len(expectedBundleHash) != sha256.Size {
		return fmt.Errorf("expected bundle hash must be %d bytes, got %d", sha256.Size, len(expectedBundleHash))
	}

	storedHash, err := hex.DecodeString(strings.TrimSpace(signature.BundleHash))
	if err != nil {
		return fmt.Errorf("decode bundle_hash: %w", err)
	}
	if len(storedHash) != sha256.Size {
		return fmt.Errorf("decode bundle_hash: invalid SHA-256 length")
	}
	if !bytes.Equal(storedHash, expectedBundleHash) {
		return fmt.Errorf(
			"bundle_hash mismatch: signature=%s snapshot=%s",
			hex.EncodeToString(storedHash),
			hex.EncodeToString(expectedBundleHash),
		)
	}

	algo := signature.Algorithm
	if algo == "" {
		algo = "ed25519"
	}

	rawSignature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature.Signature))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	rawPublicKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature.PublicKey))
	if err != nil {
		return fmt.Errorf("decode pubkey: %w", err)
	}

	switch algo {
	case "ed25519":
		if len(rawSignature) != ed25519.SignatureSize {
			return fmt.Errorf("decode signature: invalid Ed25519 signature length")
		}
		if len(rawPublicKey) != ed25519.PublicKeySize {
			return fmt.Errorf("decode pubkey: invalid Ed25519 public key length")
		}
		if !ed25519.Verify(ed25519.PublicKey(rawPublicKey), storedHash, rawSignature) {
			return fmt.Errorf("ed25519 signature mismatch: %w", ErrTamper)
		}
		return nil
	case "ecdsa-p256":
		pub, err := parseECDSAP256PublicKey(rawPublicKey)
		if err != nil {
			return fmt.Errorf("decode pubkey: %w", err)
		}
		if !ecdsa.VerifyASN1(pub, storedHash, rawSignature) {
			return fmt.Errorf("ecdsa-p256 signature mismatch: %w", ErrTamper)
		}
		return nil
	default:
		return fmt.Errorf("sign: unsupported algorithm %q: upgrade atb to verify this bundle", algo)
	}
}

// parseECDSAP256PublicKey accepts either a PKIX DER-encoded public key
// (preferred for KMS backends) or a raw 65-byte uncompressed point
// (0x04 || X || Y). Both forms are produced in the wild by different
// KMS signers, so we try DER first and fall back to raw.
func parseECDSAP256PublicKey(raw []byte) (*ecdsa.PublicKey, error) {
	if pub, err := x509.ParsePKIXPublicKey(raw); err == nil {
		ecPub, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("PKIX public key is not ECDSA")
		}
		if ecPub.Curve != elliptic.P256() {
			return nil, fmt.Errorf("ECDSA public key curve is not P-256")
		}
		return ecPub, nil
	}
	if len(raw) == 65 && raw[0] == 0x04 {
		//lint:ignore SA1019 SEC1 uncompressed is the documented wire format for KMS-emitted P-256 keys; crypto/ecdh is ECDH-only and cannot reach the ecdsa.PublicKey shape
		x, y := elliptic.Unmarshal(elliptic.P256(), raw)
		if x == nil || y == nil {
			return nil, fmt.Errorf("invalid P-256 uncompressed point")
		}
		return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil
	}
	return nil, fmt.Errorf("public key is neither PKIX DER nor 65-byte uncompressed P-256 point")
}

func bundleSignatureAlgorithm(fields map[string]any) string {
	algorithm := bundleSignatureField(fields, "algorithm")
	if algorithm == "" {
		return "ed25519"
	}
	return strings.ToLower(algorithm)
}

func bundleSignatureField(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	value, ok := fields[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
