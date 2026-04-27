// SPDX-License-Identifier: MIT

package signer

import (
	"context"
	"strings"
	"testing"
)

type registryTestSigner struct{}

func (registryTestSigner) Sign(context.Context, []byte) ([]byte, []byte, string, string, string, error) {
	return []byte("sig"), []byte("pub"), "key", "test-backend", "test-algorithm", nil
}

func TestResolveUnknownBackendMentionsBuildTags(t *testing.T) {
	_, err := Resolve(context.Background(), "missing-backend", "key")
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
	if !strings.Contains(err.Error(), "was the binary built with -tags") {
		t.Fatalf("error = %q, want build-tags hint", err.Error())
	}
}

func TestRegisterFactoryIsCallable(t *testing.T) {
	name := "registry-test"
	t.Cleanup(func() {
		delete(backends, name)
	})

	Register(name, func(_ context.Context, keyID string) (Signer, error) {
		if keyID != "key-1" {
			t.Fatalf("keyID = %q, want key-1", keyID)
		}
		return registryTestSigner{}, nil
	})

	s, err := Resolve(context.Background(), name, "key-1")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	_, _, keyID, backend, algorithm, err := s.Sign(context.Background(), []byte("digest"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if keyID != "key" || backend != "test-backend" || algorithm != "test-algorithm" {
		t.Fatalf("unexpected signer result: keyID=%q backend=%q algorithm=%q", keyID, backend, algorithm)
	}
}
