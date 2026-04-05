package sign

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

type BundleSignature struct {
	BundleHash string
	Signature  string
	PublicKey  string
}

func NewBundleSignatureRecord(bundleHash []byte, publicKey ed25519.PublicKey, signature []byte) map[string]string {
	return map[string]string{
		"bundle_hash": hex.EncodeToString(bundleHash),
		"signature":   base64.StdEncoding.EncodeToString(signature),
		"pubkey":      base64.StdEncoding.EncodeToString(publicKey),
	}
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

	rawSignature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature.Signature))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(rawSignature) != ed25519.SignatureSize {
		return fmt.Errorf("decode signature: invalid Ed25519 signature length")
	}

	rawPublicKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature.PublicKey))
	if err != nil {
		return fmt.Errorf("decode pubkey: %w", err)
	}
	if len(rawPublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("decode pubkey: invalid Ed25519 public key length")
	}

	if !ed25519.Verify(ed25519.PublicKey(rawPublicKey), storedHash, rawSignature) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
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
