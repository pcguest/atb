// SPDX-License-Identifier: MIT

//go:build awskms

package awskms

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/pcguest/atb/internal/signer"
)

// AWSKMSSigner implements internal/signer.Signer using AWS KMS.
// The KMS key must be an asymmetric signing key.
//
// NOTE: AWS KMS does not natively support Ed25519 as of 2026. AWSKMSSigner
// therefore signs with ECDSA_SHA_256 (P-256) by default and records
// algorithm="ecdsa-p256" in the signature record. The ATB verifier must be
// extended to support ECDSA verification before this signer is usable in
// production. For now the signer is implemented and tested against a mock KMS
// client; the CLI wires it in and warns that ecdsa-p256 verification is not yet
// implemented.
type AWSKMSSigner struct {
	client *kms.Client
	keyID  string

	mu     sync.Mutex
	pubKey []byte
}

func init() {
	signer.Register("aws-kms", func(ctx context.Context, keyID string) (signer.Signer, error) {
		return New(ctx, keyID)
	})
}

// New loads AWS credentials from the standard AWS SDK chain and returns an
// AWSKMSSigner for keyID.
func New(ctx context.Context, keyID string) (*AWSKMSSigner, error) {
	if keyID == "" {
		return nil, fmt.Errorf("signer: aws-kms: keyID is required")
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("signer: aws-kms: load config: %w", err)
	}
	return &AWSKMSSigner{
		client: kms.NewFromConfig(cfg),
		keyID:  keyID,
	}, nil
}

// Sign implements signer.Signer. digest is the 32-byte SHA-256 of the
// pre-signature bundle. AWS KMS Sign takes a raw digest; MessageType is DIGEST.
func (s *AWSKMSSigner) Sign(ctx context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error) {
	if s == nil || s.client == nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: aws-kms: nil client")
	}
	out, err := s.client.Sign(ctx, &kms.SignInput{
		KeyId:            aws.String(s.keyID),
		Message:          append([]byte(nil), digest...),
		MessageType:      types.MessageTypeDigest,
		SigningAlgorithm: types.SigningAlgorithmSpecEcdsaSha256,
	})
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: aws-kms: sign: %w", err)
	}
	pubKey, err = s.cachedPublicKey(ctx)
	if err != nil {
		return nil, nil, "", "", "", err
	}
	return append([]byte(nil), out.Signature...), pubKey, s.keyID, "aws-kms", "ecdsa-p256", nil
}

func (s *AWSKMSSigner) cachedPublicKey(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pubKey) > 0 {
		return append([]byte(nil), s.pubKey...), nil
	}
	out, err := s.client.GetPublicKey(ctx, &kms.GetPublicKeyInput{
		KeyId: aws.String(s.keyID),
	})
	if err != nil {
		return nil, fmt.Errorf("signer: aws-kms: get public key: %w", err)
	}
	pubKey, err := parseP256SPKI(out.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("signer: aws-kms: parse public key: %w", err)
	}
	s.pubKey = pubKey
	return append([]byte(nil), s.pubKey...), nil
}

func parseP256SPKI(der []byte) ([]byte, error) {
	parsed, err := x509.ParsePKIXPublicKey(der)
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
	//lint:ignore SA1019 SEC1 uncompressed encoding is the documented AWS KMS P-256 wire format; crypto/ecdh is ECDH-only and cannot serialise an ecdsa.PublicKey
	raw := elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)
	if len(raw) != 65 {
		return nil, fmt.Errorf("uncompressed public key length = %d, want 65", len(raw))
	}
	return raw, nil
}

var _ signer.Signer = (*AWSKMSSigner)(nil)
