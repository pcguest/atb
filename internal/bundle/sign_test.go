// SPDX-License-Identifier: MIT
package bundle

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSignToWithSignerPreservesBundleWhenAtomicWriteFails(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 keypair: %v", err)
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"ok": true}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle before signing: %v", err)
	}

	writeErr := errors.New("forced atomic write failure")
	previousAtomicWrite := atomicWrite
	atomicWrite = func(string, []byte, os.FileMode) error {
		return writeErr
	}
	t.Cleanup(func() {
		atomicWrite = previousAtomicWrite
	})

	if err := Sign(path, privateKey); !errors.Is(err, writeErr) {
		t.Fatalf("Sign() error = %v, want %v", err, writeErr)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle after failed signing: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("bundle changed after failed atomic write")
	}
}
