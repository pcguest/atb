// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestRunKeygen(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "keygen_writes_both_files",
			run: func(t *testing.T) {
				dir := t.TempDir()

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				exitCode := runKeygen([]string{"--out-dir", dir}, &stdout, &stderr)
				if exitCode != exitSuccess {
					t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
				}

				privateKeyPath := filepath.Join(dir, "atb-key.pem")
				publicKeyPath := filepath.Join(dir, "atb-key.pub.pem")
				privatePEM, err := os.ReadFile(privateKeyPath)
				if err != nil {
					t.Fatalf("read private key: %v", err)
				}
				publicPEM, err := os.ReadFile(publicKeyPath)
				if err != nil {
					t.Fatalf("read public key: %v", err)
				}

				privateBlock, _ := pem.Decode(privatePEM)
				if privateBlock == nil || len(privateBlock.Bytes) == 0 {
					t.Fatalf("expected non-empty private key PEM block")
				}
				publicBlock, _ := pem.Decode(publicPEM)
				if publicBlock == nil || len(publicBlock.Bytes) == 0 {
					t.Fatalf("expected non-empty public key PEM block")
				}
			},
		},
		{
			name: "keygen_refuses_overwrite",
			run: func(t *testing.T) {
				dir := t.TempDir()

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				if exitCode := runKeygen([]string{"--out-dir", dir}, &stdout, &stderr); exitCode != exitSuccess {
					t.Fatalf("unexpected first exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
				}

				stdout.Reset()
				stderr.Reset()
				exitCode := runKeygen([]string{"--out-dir", dir}, &stdout, &stderr)
				if exitCode == exitSuccess {
					t.Fatalf("expected second keygen to fail")
				}
			},
		},
		{
			name: "keygen_key_is_usable",
			run: func(t *testing.T) {
				dir := t.TempDir()

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				exitCode := runKeygen([]string{"--out-dir", dir}, &stdout, &stderr)
				if exitCode != exitSuccess {
					t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
				}

				privateKey, err := loadEd25519PrivateKey(filepath.Join(dir, "atb-key.pem"))
				if err != nil {
					t.Fatalf("load private key: %v", err)
				}

				publicPEM, err := os.ReadFile(filepath.Join(dir, "atb-key.pub.pem"))
				if err != nil {
					t.Fatalf("read public key: %v", err)
				}
				block, _ := pem.Decode(publicPEM)
				if block == nil {
					t.Fatalf("expected public key PEM block")
				}
				parsedPublicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					t.Fatalf("parse public key: %v", err)
				}
				publicKey, ok := parsedPublicKey.(ed25519.PublicKey)
				if !ok {
					t.Fatalf("expected Ed25519 public key, got %T", parsedPublicKey)
				}

				message := []byte("atb-keygen-test-message")
				signature := ed25519.Sign(privateKey, message)
				if !ed25519.Verify(publicKey, message, signature) {
					t.Fatalf("expected signature verification to pass")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestRunKeygenWarnsAtRepoRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exit := runKeygen([]string{"--out-dir", dir}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("repository root")) {
		t.Fatalf("stderr did not contain repo-root warning: %q", stderr.String())
	}

	info, err := os.Stat(filepath.Join(dir, "atb-key.pem"))
	if err != nil {
		t.Fatalf("stat private key: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("private key mode = %o, want 0600", info.Mode().Perm())
	}
}
