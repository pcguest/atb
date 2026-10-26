// SPDX-License-Identifier: MIT
package bundle

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	signpkg "github.com/pcguest/atb/v2/internal/sign"
	"github.com/pcguest/atb/v2/internal/signer"
)

func TestSigningReadBound(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		size      int
		wantLimit bool
	}{
		{name: "below", size: 7},
		{name: "at", size: 8},
		{name: "plus-one", size: 9, wantLimit: true},
		{name: "large-source-not-drained", size: 100, wantLimit: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(strings.Repeat("x", tc.size))
			got, err := readSigningInput(r, 8)
			if errors.Is(err, ErrResourceLimit) != tc.wantLimit || (err != nil && !tc.wantLimit) {
				t.Fatalf("read error = %v, want limit=%v", err, tc.wantLimit)
			}
			if !tc.wantLimit && string(got) != strings.Repeat("x", tc.size) {
				t.Fatal("accepted input bytes changed")
			}
			if consumed, want := tc.size-r.Len(), min(tc.size, 9); consumed != want {
				t.Fatalf("consumed %d bytes, want %d (at most limit plus one)", consumed, want)
			}
		})
	}
	for _, limit := range []int64{-1, 0, math.MaxInt64} {
		if _, err := readSigningInput(strings.NewReader("x"), limit); !errors.Is(err, ErrResourceLimit) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
	wantErr := errors.New("source failed")
	if _, err := readSigningInput(failingSigningReader{err: wantErr}, 8); !errors.Is(err, wantErr) {
		t.Fatalf("source error not preserved: %v", err)
	}
}

type failingSigningReader struct{ err error }

func (r failingSigningReader) Read([]byte) (int, error) { return 0, r.err }

type countingSigningSigner struct {
	signer.Signer
	calls int
}

func (s *countingSigningSigner) Sign(ctx context.Context, digest []byte) ([]byte, []byte, string, string, string, error) {
	s.calls++
	return s.Signer.Sign(ctx, digest)
}

func signingFixture(t *testing.T, terminated bool) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.atb")
	b, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.TrimSuffix(raw, []byte("\n"))
	if terminated {
		raw = append(raw, '\n')
	}
	return raw
}

func signedOutputSizeFor(t *testing.T, raw []byte) int {
	t.Helper()
	dir := t.TempDir()
	input := filepath.Join(dir, "input.atb")
	output := filepath.Join(dir, "output.atb")
	if err := os.WriteFile(input, raw, 0600); err != nil {
		t.Fatal(err)
	}
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	s := &countingSigningSigner{Signer: signer.NewLocalSigner(key)}
	if _, err := signToWithSignerLimited(context.Background(), input, output, s, MaxBundleSizeBytes); err != nil {
		t.Fatal(err)
	}
	signed, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return len(signed)
}

func TestSigningLimitRejectsBeforeSignerAndWrite(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, mode string
		size       int
	}{
		{name: "in-place", mode: "in-place", size: 64},
		{name: "existing-output", mode: "existing-output", size: 64},
		{name: "missing-output", mode: "missing-output", size: 64},
		{name: "in-place-plus-one", mode: "in-place", size: 9},
		{name: "existing-output-plus-one", mode: "existing-output", size: 9},
		{name: "missing-output-plus-one", mode: "missing-output", size: 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.atb")
			before := []byte(strings.Repeat("x", tc.size))
			if err := os.WriteFile(input, before, 0600); err != nil {
				t.Fatal(err)
			}
			output := input
			if tc.mode != "in-place" {
				output = filepath.Join(dir, "output.atb")
			}
			if tc.mode == "existing-output" {
				if err := os.WriteFile(output, []byte("preserve output"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			// A nil delegate intentionally fails if rejected input ever reaches Sign.
			s := &countingSigningSigner{}
			_, err := signToWithSignerLimited(context.Background(), input, output, s, 8)
			if !errors.Is(err, ErrResourceLimit) || s.calls != 0 {
				t.Fatalf("err=%v, signer calls=%d", err, s.calls)
			}
			after, err := os.ReadFile(input)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("input changed: %v", err)
			}
			if tc.mode == "existing-output" {
				after, err = os.ReadFile(output)
				if err != nil || string(after) != "preserve output" {
					t.Fatalf("output changed: %v", err)
				}
			}
			if tc.mode == "missing-output" {
				if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("output created: %v", err)
				}
			}
		})
	}
}

func TestSigningAtLimitPreservesPrefixAndSignature(t *testing.T) {
	t.Parallel()
	for _, newline := range []bool{false, true} {
		name := "without-newline"
		if newline {
			name = "with-newline"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bundle.atb")
			b, err := New()
			if err != nil {
				t.Fatal(err)
			}
			if err := b.Append("ai.tool.exec", map[string]any{"ok": true}); err != nil {
				t.Fatal(err)
			}
			if err := b.Save(path); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			raw = bytes.TrimSuffix(raw, []byte("\n"))
			if newline {
				raw = append(raw, '\n')
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
			s := &countingSigningSigner{Signer: signer.NewLocalSigner(key)}
			// The injected policy now bounds the emitted bundle as well as the
			// input read. The exact input-read boundary is covered separately.
			limit := int64(signedOutputSizeFor(t, raw))
			if _, err := signToWithSignerLimited(context.Background(), path, path, s, limit); err != nil {
				t.Fatal(err)
			}
			if s.calls != 1 {
				t.Fatalf("signer calls=%d", s.calls)
			}
			signed, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(signed, raw) {
				t.Fatal("original prefix changed")
			}
			tail := signed[len(raw):]
			if !newline {
				if len(tail) == 0 || tail[0] != '\n' {
					t.Fatal("missing separator newline")
				}
				tail = tail[1:]
			}
			if len(tail) == 0 || tail[0] != '{' || tail[len(tail)-1] != '\n' || bytes.Count(tail, []byte("\n")) != 1 {
				t.Fatalf("unexpected signature-record newline framing: %q", tail)
			}
			loaded, err := LoadVerified(path)
			if err != nil {
				t.Fatal(err)
			}
			sig, err := signpkg.ParseBundleSignature(loaded.Records[len(loaded.Records)-1].Event.Data)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(raw)
			if err := signpkg.VerifyBundleSignature(sig, digest[:]); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSigningOutputLimitBoundary(t *testing.T) {
	t.Parallel()
	for _, terminated := range []bool{true, false} {
		ending := "unterminated"
		if terminated {
			ending = "terminated"
		}
		raw := signingFixture(t, terminated)
		emitted := signedOutputSizeFor(t, raw)
		for _, tc := range []struct {
			name  string
			delta int
			mode  string
		}{
			{name: "below", delta: 1, mode: "in-place"},
			{name: "exact", mode: "in-place"},
			{name: "over", delta: -1, mode: "in-place"},
			{name: "below", delta: 1, mode: "existing-output"},
			{name: "exact", mode: "existing-output"},
			{name: "over", delta: -1, mode: "existing-output"},
			{name: "below", delta: 1, mode: "missing-output"},
			{name: "exact", mode: "missing-output"},
			{name: "over", delta: -1, mode: "missing-output"},
		} {
			tc := tc
			t.Run(ending+"/"+tc.name+"/"+tc.mode, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				input := filepath.Join(dir, "input.atb")
				before := bytes.Clone(raw)
				if err := os.WriteFile(input, before, 0600); err != nil {
					t.Fatal(err)
				}
				output := input
				if tc.mode != "in-place" {
					output = filepath.Join(dir, "output.atb")
				}
				if tc.mode == "existing-output" {
					if err := os.WriteFile(output, []byte("preserve output"), 0600); err != nil {
						t.Fatal(err)
					}
				}

				key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
				s := &countingSigningSigner{Signer: signer.NewLocalSigner(key)}
				_, err := signToWithSignerLimited(context.Background(), input, output, s, int64(emitted+tc.delta))
				if tc.delta < 0 {
					if !errors.Is(err, ErrResourceLimit) || !strings.Contains(err.Error(), "output") {
						t.Fatalf("error = %v, want output ErrResourceLimit", err)
					}
					if s.calls != 1 {
						t.Fatalf("signer calls = %d, want 1 after output-size rejection", s.calls)
					}
					after, readErr := os.ReadFile(input)
					if readErr != nil || !bytes.Equal(before, after) {
						t.Fatalf("input changed: %v", readErr)
					}
					switch tc.mode {
					case "existing-output":
						after, readErr = os.ReadFile(output)
						if readErr != nil || string(after) != "preserve output" {
							t.Fatalf("existing output changed: %v", readErr)
						}
					case "missing-output":
						if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
							t.Fatalf("missing output created: %v", statErr)
						}
					}
					return
				}

				if err != nil {
					t.Fatalf("sign: %v", err)
				}
				if s.calls != 1 {
					t.Fatalf("signer calls = %d, want 1", s.calls)
				}
				signed, readErr := os.ReadFile(output)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if len(signed) != emitted {
					t.Fatalf("output length = %d, want %d (limit %d)", len(signed), emitted, emitted+tc.delta)
				}
				if !bytes.HasPrefix(signed, before) {
					t.Fatal("original prefix changed")
				}
				if tc.mode != "in-place" {
					after, readErr := os.ReadFile(input)
					if readErr != nil || !bytes.Equal(before, after) {
						t.Fatalf("separate input changed: %v", readErr)
					}
				}
			})
		}
	}
}

func TestSigningAtLimitReachesParser(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "malformed.atb")
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &countingSigningSigner{}
	_, err := signToWithSignerLimited(context.Background(), path, path, s, 8)
	if !errors.Is(err, ErrMalformed) || s.calls != 0 {
		t.Fatalf("expected parse rejection at exact read limit, got %v, calls=%d", err, s.calls)
	}
}

func TestSignedPayloadMetrics(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                   string
		raw, record            int
		separator              bool
		wantSize, wantCapacity int
		wantErr                bool
	}{
		{name: "empty", record: 1, wantSize: 2, wantCapacity: 3},
		{name: "terminated", raw: 5, record: 7, wantSize: 13, wantCapacity: 14},
		{name: "unterminated", raw: 5, record: 7, separator: true, wantSize: 14, wantCapacity: 14},
		{name: "empty-separator-ignored", record: 7, separator: true, wantSize: 8, wantCapacity: 9},
		{name: "exact-max-no-separator", raw: math.MaxInt - 3, record: 1, wantSize: math.MaxInt - 1, wantCapacity: math.MaxInt},
		{name: "exact-max-separator", raw: math.MaxInt - 3, record: 1, separator: true, wantSize: math.MaxInt, wantCapacity: math.MaxInt},
		{name: "sum-overflow", raw: math.MaxInt - 3, record: 2, wantErr: true},
		{name: "newline-overflow", raw: math.MaxInt - 1, wantErr: true},
		{name: "negative-raw", raw: -1, wantErr: true},
		{name: "negative-record", record: -1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotSize, gotCapacity, err := signedPayloadMetrics(tc.raw, tc.record, tc.separator)
			if errors.Is(err, ErrResourceLimit) != tc.wantErr {
				t.Fatalf("err = %v, want limit=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if err != nil || gotSize != tc.wantSize || gotCapacity != tc.wantCapacity {
				t.Fatalf("size=%d capacity=%d err=%v, want size=%d capacity=%d", gotSize, gotCapacity, err, tc.wantSize, tc.wantCapacity)
			}
		})
	}
}

func TestSignedPayloadCapacity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              string
		raw, record, want int
		wantErr           bool
	}{
		{name: "empty", want: 2},
		{name: "normal", raw: 5, record: 7, want: 14},
		{name: "exact-max", raw: math.MaxInt - 3, record: 1, want: math.MaxInt},
		{name: "record-max", record: math.MaxInt - 2, want: math.MaxInt},
		{name: "sum-overflow", raw: math.MaxInt - 3, record: 2, wantErr: true},
		{name: "newline-overflow", raw: math.MaxInt - 1, wantErr: true},
		{name: "both-max", raw: math.MaxInt, record: math.MaxInt, wantErr: true},
		{name: "negative-raw", raw: -1, wantErr: true},
		{name: "negative-record", record: -1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := signedPayloadCapacity(tc.raw, tc.record)
			if errors.Is(err, ErrResourceLimit) != tc.wantErr || (!tc.wantErr && (err != nil || got != tc.want)) {
				t.Fatalf("capacity=%d err=%v, want=%d limit=%v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
