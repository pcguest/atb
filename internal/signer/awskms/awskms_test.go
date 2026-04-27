// SPDX-License-Identifier: MIT

//go:build awskms

package awskms

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

func TestAWSKMSSignerSign(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}

	var signCalls int
	var getPublicKeyCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		switch {
		case strings.HasSuffix(target, ".Sign"):
			signCalls++
			var req struct {
				Message string
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode sign request: %v", err)
			}
			digest, err := base64.StdEncoding.DecodeString(req.Message)
			if err != nil {
				t.Fatalf("decode digest: %v", err)
			}
			signature, err := ecdsa.SignASN1(rand.Reader, priv, digest)
			if err != nil {
				t.Fatalf("SignASN1: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"Signature": base64.StdEncoding.EncodeToString(signature),
			})
		case strings.HasSuffix(target, ".GetPublicKey"):
			getPublicKeyCalls++
			_ = json.NewEncoder(w).Encode(map[string]string{
				"PublicKey": base64.StdEncoding.EncodeToString(der),
			})
		default:
			t.Fatalf("unexpected X-Amz-Target %q", target)
		}
	}))
	defer srv.Close()

	client := kms.New(kms.Options{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("access", "secret", ""),
		HTTPClient:   srv.Client(),
		BaseEndpoint: aws.String(srv.URL),
	})
	s := &AWSKMSSigner{
		client: client,
		keyID:  "alias/test",
	}
	digest := sha256.Sum256([]byte("bundle"))

	signature, pubKey, keyID, backend, algorithm, err := s.Sign(context.Background(), digest[:])
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(signature) == 0 {
		t.Fatal("signature is empty")
	}
	if len(pubKey) != 65 {
		t.Fatalf("pubKey length = %d, want 65", len(pubKey))
	}
	if keyID != "alias/test" || backend != "aws-kms" || algorithm != "ecdsa-p256" {
		t.Fatalf("unexpected provenance: keyID=%q backend=%q algorithm=%q", keyID, backend, algorithm)
	}
	if signCalls != 1 || getPublicKeyCalls != 1 {
		t.Fatalf("call counts: sign=%d getPublicKey=%d", signCalls, getPublicKeyCalls)
	}
}
