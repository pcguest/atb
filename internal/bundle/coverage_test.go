// SPDX-License-Identifier: MIT
package bundle

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSignArgumentCompatibilityContracts(t *testing.T) {
	key := ed25519.PrivateKey(make([]byte, ed25519.PrivateKeySize))
	if ctx, path, gotKey, err := signArgs([]any{"bundle.atb", key}); err != nil ||
		ctx == nil || path != "bundle.atb" || len(gotKey) != ed25519.PrivateKeySize {
		t.Fatalf("legacy sign args ctx=%v path=%q key=%d err=%v", ctx, path, len(gotKey), err)
	}
	if ctx, path, _, err := signArgs([]any{context.Background(), "bundle.atb", key}); err != nil ||
		ctx == nil || path != "bundle.atb" {
		t.Fatalf("nil-context sign args ctx=%v path=%q err=%v", ctx, path, err)
	}
	for _, args := range [][]any{
		nil,
		{1, key},
		{"bundle.atb", "not-key"},
		{"not-context", "bundle.atb", key},
		{context.Background(), 1, key},
		{context.Background(), "bundle.atb", "not-key"},
	} {
		if _, _, _, err := signArgs(args); err == nil {
			t.Fatalf("signArgs(%#v) succeeded", args)
		}
	}

	if ctx, in, out, gotKey, err := signToArgs([]any{"in.atb", "out.atb", key}); err != nil ||
		ctx == nil || in != "in.atb" || out != "out.atb" || len(gotKey) != ed25519.PrivateKeySize {
		t.Fatalf("legacy sign-to args ctx=%v in=%q out=%q key=%d err=%v", ctx, in, out, len(gotKey), err)
	}
	if ctx, in, out, _, err := signToArgs([]any{context.Background(), "in.atb", "out.atb", key}); err != nil ||
		ctx == nil || in != "in.atb" || out != "out.atb" {
		t.Fatalf("nil-context sign-to args ctx=%v in=%q out=%q err=%v", ctx, in, out, err)
	}
	for _, args := range [][]any{
		nil,
		{1, "out.atb", key},
		{"in.atb", 1, key},
		{"in.atb", "out.atb", "not-key"},
		{"not-context", "in.atb", "out.atb", key},
		{context.Background(), 1, "out.atb", key},
		{context.Background(), "in.atb", 1, key},
		{context.Background(), "in.atb", "out.atb", "not-key"},
	} {
		if _, _, _, _, err := signToArgs(args); err == nil {
			t.Fatalf("signToArgs(%#v) succeeded", args)
		}
	}
}

func TestSaveWithLockContracts(t *testing.T) {
	b, err := New()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bundle.atb")
	released := false
	err = b.saveWithLock(context.Background(), path, func(got string) (func() error, error) {
		if got != path {
			t.Fatalf("acquire path=%q", got)
		}
		return func() error {
			released = true
			return nil
		}, nil
	})
	if err != nil || !released {
		t.Fatalf("save err=%v released=%v", err, released)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved bundle: %v", err)
	}

	acquireErr := errors.New("lock failed")
	err = b.saveWithLock(context.Background(), path, func(string) (func() error, error) {
		return nil, acquireErr
	})
	if !errors.Is(err, acquireErr) {
		t.Fatalf("acquire error=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.SaveWithRetry(ctx, path, time.Second, time.Millisecond); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled save error=%v", err)
	}
}

func TestManifestVersionHelperContracts(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  int
	}{
		{value: nil, want: 1},
		{value: float64(2), want: 2},
		{value: 2, want: 2},
		{value: "", want: 1},
		{value: "2", want: 2},
	} {
		got, err := manifestVersionToInt(tc.value)
		if err != nil || got != tc.want {
			t.Fatalf("manifestVersionToInt(%#v)=%d err=%v", tc.value, got, err)
		}
	}
	for _, value := range []any{"not-int", true} {
		if _, err := manifestVersionToInt(value); err == nil {
			t.Fatalf("manifestVersionToInt(%#v) succeeded", value)
		}
	}

	for _, value := range []any{float64(2), 2, "2"} {
		if got, err := manifestVersionString(value); err != nil || got != "2" {
			t.Fatalf("manifestVersionString(%#v)=%q err=%v", value, got, err)
		}
	}
	for _, value := range []any{1, "", nil} {
		if _, err := manifestVersionString(value); err == nil {
			t.Fatalf("manifestVersionString(%#v) succeeded", value)
		}
	}
	if got := DefaultPath(); !strings.HasSuffix(got, filepath.Join(BundleDir, BundleFile)) {
		t.Fatalf("DefaultPath=%q", got)
	}
}
