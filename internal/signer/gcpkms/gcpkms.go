// SPDX-License-Identifier: MIT

//go:build gcpkms

package gcpkms

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/googleapis/gax-go/v2"
	"github.com/pcguest/atb/internal/signer"
)

type cloudKMSClient interface {
	AsymmetricSign(context.Context, *kmspb.AsymmetricSignRequest, ...gax.CallOption) (*kmspb.AsymmetricSignResponse, error)
	GetPublicKey(context.Context, *kmspb.GetPublicKeyRequest, ...gax.CallOption) (*kmspb.PublicKey, error)
}

// GCPKMSSigner implements internal/signer.Signer using GCP Cloud KMS.
//
// NOTE: ATB records GCP KMS signatures as algorithm="ecdsa-p256". The ATB
// verifier supports ECDSA P-256 using the public key embedded in the signature
// record, so verification remains offline and does not require GCP credentials.
type GCPKMSSigner struct {
	client cloudKMSClient
	keyID  string

	mu     sync.Mutex
	pubKey []byte
}

func init() {
	signer.Register("gcp-kms", func(ctx context.Context, keyID string) (signer.Signer, error) {
		return New(ctx, keyID)
	})
}

// New loads GCP credentials from the standard Application Default Credentials
// chain and returns a GCPKMSSigner for keyID.
func New(ctx context.Context, keyID string) (*GCPKMSSigner, error) {
	if keyID == "" {
		return nil, fmt.Errorf("signer: gcp-kms: keyID is required")
	}
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("signer: gcp-kms: new client: %w", err)
	}
	return &GCPKMSSigner{
		client: client,
		keyID:  keyID,
	}, nil
}

// Sign implements signer.Signer. digest is the 32-byte SHA-256 of the
// pre-signature bundle.
func (s *GCPKMSSigner) Sign(ctx context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error) {
	if s == nil || s.client == nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: gcp-kms: nil client")
	}
	out, err := s.client.AsymmetricSign(ctx, &kmspb.AsymmetricSignRequest{
		Name: s.keyID,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{Sha256: append([]byte(nil), digest...)},
		},
	})
	if err != nil {
		return nil, nil, "", "", "", fmt.Errorf("signer: gcp-kms: sign: %w", err)
	}
	pubKey, err = s.cachedPublicKey(ctx)
	if err != nil {
		return nil, nil, "", "", "", err
	}
	return append([]byte(nil), out.Signature...), pubKey, s.keyID, "gcp-kms", "ecdsa-p256", nil
}

func (s *GCPKMSSigner) cachedPublicKey(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pubKey) > 0 {
		return append([]byte(nil), s.pubKey...), nil
	}
	out, err := s.client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{
		Name: s.keyID,
	})
	if err != nil {
		return nil, fmt.Errorf("signer: gcp-kms: get public key: %w", err)
	}
	pubKey, err := parseP256PEM(out.Pem)
	if err != nil {
		return nil, fmt.Errorf("signer: gcp-kms: parse public key: %w", err)
	}
	s.pubKey = pubKey
	return append([]byte(nil), s.pubKey...), nil
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
	//lint:ignore SA1019 SEC1 uncompressed encoding is the documented GCP Cloud KMS P-256 wire format; crypto/ecdh is ECDH-only and cannot serialise an ecdsa.PublicKey
	raw := elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)
	if len(raw) != 65 {
		return nil, fmt.Errorf("uncompressed public key length = %d, want 65", len(raw))
	}
	return raw, nil
}

var _ signer.Signer = (*GCPKMSSigner)(nil)
