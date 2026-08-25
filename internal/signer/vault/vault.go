// SPDX-License-Identifier: MIT

//go:build vault

package vault

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/pcguest/atb/internal/signer"
)

const (
	keyTypeEd25519   = "ed25519"
	keyTypeECDSAP256 = "ecdsa-p256"
)

type vaultLogical interface {
	WriteWithContext(context.Context, string, map[string]any) (*vaultapi.Secret, error)
	ReadWithContext(context.Context, string) (*vaultapi.Secret, error)
}

// VaultSigner implements internal/signer.Signer using the Vault transit engine.
//
// The signer reads the key type from /v1/transit/keys/<name> and selects the
// algorithm accordingly: "ed25519" or "ecdsa-p256". Other Vault key types
// (RSA, larger ECDSA curves) are rejected. If the type cannot be retrieved,
// the signer falls back to "ecdsa-p256" and emits a warning so signing still
// succeeds with documented provenance.
type VaultSigner struct {
	client  *vaultapi.Client
	logical vaultLogical
	keyID   string

	// stderr receives the fallback warning when key-type detection fails.
	// Tests can override; nil disables the warning.
	stderr io.Writer

	mu      sync.Mutex
	pubKey  []byte
	keyType string
}

func init() {
	signer.Register("vault", func(ctx context.Context, keyID string) (signer.Signer, error) {
		return New(ctx, keyID)
	})
}

// New loads VAULT_ADDR and VAULT_TOKEN from the environment and returns a
// VaultSigner for the transit key name in keyID.
func New(_ context.Context, keyID string) (*VaultSigner, error) {
	if keyID == "" {
		return nil, fmt.Errorf("signer: vault: keyID is required")
	}
	cfg := vaultapi.DefaultConfig()
	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("signer: vault: new client: %w", err)
	}
	return &VaultSigner{
		client:  client,
		logical: client.Logical(),
		keyID:   keyID,
		stderr:  os.Stderr,
	}, nil
}

// Sign implements signer.Signer. digest is the 32-byte SHA-256 of the
// pre-signature bundle.
func (s *VaultSigner) Sign(ctx context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error) {
	if s == nil || s.logical == nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: vault: nil client")
	}
	keyType, pubKey, err := s.cachedKeyMaterial(ctx)
	if err != nil {
		return nil, nil, "", "", "", err
	}

	path := "transit/sign/" + strings.TrimPrefix(s.keyID, "/")
	params := map[string]any{
		"input": base64.StdEncoding.EncodeToString(digest),
	}
	switch keyType {
	case keyTypeECDSAP256:
		params["prehashed"] = true
		params["hash_algorithm"] = "sha2-256"
		params["marshaling_algorithm"] = "raw"
	case keyTypeEd25519:
		// Vault Ed25519 only supports prehashed=false; the caller's digest is
		// signed as the message body. The verifier must use the same input.
		params["prehashed"] = false
	default:
		return nil, nil, "", "", "", fmt.Errorf("signer: vault: unsupported key type %q", keyType)
	}

	secret, err := s.logical.WriteWithContext(ctx, path, params)
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: vault: sign: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: vault: empty sign response")
	}
	signatureText, _ := secret.Data["signature"].(string)
	signature, err := decodeVaultSignature(signatureText)
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: vault: decode signature: %w", err)
	}
	return signature, pubKey, s.keyID, "vault", keyType, nil
}

// cachedKeyMaterial reads the Vault Transit key once and caches the type
// and the parsed public key. If the read response omits the type field
// (older Vault API or proxied client), the signer falls back to ecdsa-p256
// and warns once on stderr — signing still completes with documented
// provenance, but operators should upgrade Vault to expose the type.
//
// The ecdsa-p256 fallback preserves compatibility with Vault versions or
// proxies that omit "type" on /v1/transit/keys/<name>.
func (s *VaultSigner) cachedKeyMaterial(ctx context.Context) (string, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.keyType != "" && len(s.pubKey) > 0 {
		return s.keyType, append([]byte(nil), s.pubKey...), nil
	}

	path := "transit/keys/" + strings.TrimPrefix(s.keyID, "/")
	secret, err := s.logical.ReadWithContext(ctx, path)
	if err != nil {
		return "", nil, fmt.Errorf("signer: vault: read key: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", nil, fmt.Errorf("signer: vault: empty key response")
	}

	keyType, _ := secret.Data["type"].(string)
	keyType = strings.TrimSpace(strings.ToLower(keyType))
	if keyType == "" {
		if s.stderr != nil {
			fmt.Fprintln(s.stderr, "signer: vault: key type unavailable; falling back to ecdsa-p256")
		}
		keyType = keyTypeECDSAP256
	}

	pubRaw, err := vaultPublicKeyPEM(secret.Data)
	if err != nil {
		return "", nil, fmt.Errorf("signer: vault: public key: %w", err)
	}

	var pubKey []byte
	switch keyType {
	case keyTypeECDSAP256:
		pubKey, err = parseP256PEM(pubRaw)
		if err != nil {
			return "", nil, fmt.Errorf("signer: vault: parse public key: %w", err)
		}
	case keyTypeEd25519:
		pubKey, err = parseEd25519PublicKey(pubRaw)
		if err != nil {
			return "", nil, fmt.Errorf("signer: vault: parse public key: %w", err)
		}
	default:
		return "", nil, fmt.Errorf("signer: vault: unsupported key type %q", keyType)
	}

	s.keyType = keyType
	s.pubKey = pubKey
	return s.keyType, append([]byte(nil), s.pubKey...), nil
}

// parseEd25519PublicKey accepts either a PEM-encoded PKIX Ed25519 public key
// or a bare base64-encoded 32-byte raw public key, both forms Vault has been
// observed to emit across versions.
func parseEd25519PublicKey(text string) ([]byte, error) {
	text = strings.TrimSpace(text)
	if block, _ := pem.Decode([]byte(text)); block != nil {
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		pub, ok := parsed.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("ed25519 public key parse: unexpected type %T", parsed)
		}
		if len(pub) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("ed25519 public key length = %d, want %d", len(pub), ed25519.PublicKeySize)
		}
		return []byte(pub), nil
	}
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("ed25519 public key not PEM and not base64: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("ed25519 public key length = %d, want 32", len(raw))
	}
	return raw, nil
}

func decodeVaultSignature(text string) ([]byte, error) {
	parts := strings.Split(text, ":")
	if len(parts) > 0 {
		text = parts[len(parts)-1]
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("signature absent")
	}
	return base64.StdEncoding.DecodeString(text)
}

func vaultPublicKeyPEM(data map[string]any) (string, error) {
	keys, ok := data["keys"].(map[string]any)
	if !ok || len(keys) == 0 {
		return "", fmt.Errorf("keys absent")
	}
	latest := fmt.Sprint(data["latest_version"])
	if latest == "" || latest == "<nil>" {
		for version := range keys {
			latest = version
			break
		}
	}
	entry, ok := keys[latest].(map[string]any)
	if !ok {
		return "", fmt.Errorf("key version %q absent", latest)
	}
	pubPEM, _ := entry["public_key"].(string)
	if pubPEM == "" {
		return "", fmt.Errorf("public_key absent")
	}
	return pubPEM, nil
}

func parseP256PEM(text string) ([]byte, error) {
	block, _ := pem.Decode([]byte(text))
	if block == nil {
		return nil, fmt.Errorf("public key PEM is malformed")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is %T, want *ecdsa.PublicKey", parsed)
	}
	if pub.Curve != elliptic.P256() {
		return nil, fmt.Errorf("public key curve is not P-256")
	}
	//lint:ignore SA1019 SEC1 uncompressed encoding is the documented Vault Transit P-256 wire format; crypto/ecdh is ECDH-only and cannot serialise an ecdsa.PublicKey
	raw := elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)
	if len(raw) != 65 {
		return nil, fmt.Errorf("uncompressed public key length = %d, want 65", len(raw))
	}
	return raw, nil
}

var _ signer.Signer = (*VaultSigner)(nil)
