// SPDX-License-Identifier: MIT

//go:build gcpkms

package gcpkms

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/googleapis/gax-go/v2"
)

type fakeCloudKMSClient struct {
	pem       string
	signature []byte
}

func (f fakeCloudKMSClient) AsymmetricSign(context.Context, *kmspb.AsymmetricSignRequest, ...gax.CallOption) (*kmspb.AsymmetricSignResponse, error) {
	return &kmspb.AsymmetricSignResponse{Signature: append([]byte(nil), f.signature...)}, nil
}

func (f fakeCloudKMSClient) GetPublicKey(context.Context, *kmspb.GetPublicKeyRequest, ...gax.CallOption) (*kmspb.PublicKey, error) {
	return &kmspb.PublicKey{Pem: f.pem}, nil
}

func TestGCPKMSSignerSign(t *testing.T) {
	pubPEM := testP256PublicKeyPEM(t)
	s := &GCPKMSSigner{
		client: fakeCloudKMSClient{
			pem:       pubPEM,
			signature: []byte("signature"),
		},
		keyID: "projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1",
	}

	signature, pubKey, keyID, backend, algorithm, err := s.Sign(context.Background(), make([]byte, 32))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if string(signature) != "signature" {
		t.Fatalf("signature = %q, want signature", string(signature))
	}
	if len(pubKey) != 65 {
		t.Fatalf("pubKey length = %d, want 65", len(pubKey))
	}
	if keyID != s.keyID || backend != "gcp-kms" || algorithm != "ecdsa-p256" {
		t.Fatalf("unexpected provenance: keyID=%q backend=%q algorithm=%q", keyID, backend, algorithm)
	}
}

func testP256PublicKeyPEM(t *testing.T) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}
