// SPDX-License-Identifier: MIT
package bundle_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/signer"
	"github.com/pcguest/atb/internal/verify"
)

type newlineDigestSigner struct {
	signer.Signer
	digests [][]byte
}

// Recreate the legacy wire form independently of SignTo: optional algorithm and
// provenance fields are absent, and the signature covers the original raw input.
func TestLegacyUnterminatedSignatureRequiresExactBytesAndSignature(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		separator        string
		invalidSignature bool
		wantValid        bool
	}{
		{name: "legacy-original", separator: "\n", wantValid: true},
		{name: "extra-blank-line", separator: "\n\n"},
		{name: "added-cr", separator: "\r\n"},
		{name: "added-space", separator: " \n"},
		{name: "invalid-signature", separator: "\n", invalidSignature: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "legacy.atb")
			b, err := bundle.New()
			if err != nil {
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
			digest := sha256.Sum256(raw)
			key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
			sig := ed25519.Sign(key, digest[:])
			if tc.invalidSignature {
				sig = make([]byte, ed25519.SignatureSize)
			}
			if err := b.Append("atb.bundle.signature", map[string]any{
				"bundle_hash": hex.EncodeToString(digest[:]),
				"signature":   base64.StdEncoding.EncodeToString(sig),
				"public_key":  base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)),
			}); err != nil {
				t.Fatal(err)
			}
			record, err := json.Marshal(b.Records[len(b.Records)-1])
			if err != nil {
				t.Fatal(err)
			}
			wire := append(bytes.Clone(raw), []byte(tc.separator)...)
			wire = append(wire, record...)
			wire = append(wire, '\n')
			if err := os.WriteFile(path, wire, 0600); err != nil {
				t.Fatal(err)
			}
			loaded, err := bundle.LoadVerified(path)
			if err != nil {
				t.Fatal(err)
			}
			report := verify.Verify(loaded, path, "")
			if report.BundleSignature == nil || report.BundleSignature.Verified != tc.wantValid {
				t.Fatalf("public result = %+v, want valid=%v", report.BundleSignature, tc.wantValid)
			}
			if len(report.Signatures) != 1 || report.Signatures[0].Valid != tc.wantValid {
				t.Fatalf("all-signature result = %+v, want valid=%v", report.Signatures, tc.wantValid)
			}
		})
	}
}

func (s *newlineDigestSigner) Sign(ctx context.Context, digest []byte) ([]byte, []byte, string, string, string, error) {
	s.digests = append(s.digests, bytes.Clone(digest))
	return s.Signer.Sign(ctx, digest)
}

func TestSignNewlinePublicVerification(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		ending   string
		crlfOnly bool
	}{
		{name: "lf", ending: "\n"},
		{name: "unterminated"},
		{name: "crlf", ending: "\r\n"},
		{name: "all-records-crlf", ending: "\r\n", crlfOnly: true},
		{name: "lf-with-blank-line", ending: "\n\n"},
		{name: "crlf-with-blank-line", ending: "\r\n\r\n"},
		{name: "unterminated-trailing-spaces", ending: "  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "signed.atb")
			b, err := bundle.New()
			if err != nil {
				t.Fatal(err)
			}
			if err := b.Append("ai.tool.exec", map[string]any{"synthetic": true}); err != nil {
				t.Fatal(err)
			}
			if err := b.Save(path); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			raw = append(bytes.TrimSuffix(raw, []byte("\n")), []byte(tc.ending)...)
			if tc.crlfOnly {
				raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
				raw = bytes.ReplaceAll(raw, []byte("\n"), []byte("\r\n"))
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
			s := &newlineDigestSigner{Signer: signer.NewLocalSigner(key)}
			for pass := range 2 {
				digest := sha256.Sum256(raw)
				returnedHash, err := bundle.SignToWithSigner(context.Background(), path, path, s)
				if err != nil {
					t.Fatal(err)
				}
				if returnedHash != hex.EncodeToString(digest[:]) {
					t.Fatal("returned bundle hash does not cover the exact original input")
				}
				if len(s.digests) != pass+1 || !bytes.Equal(s.digests[pass], digest[:]) {
					t.Fatal("signer invocation count or original-byte preimage changed")
				}
				signed, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.HasPrefix(signed, raw) {
					t.Fatal("original input prefix changed")
				}
				loaded, err := bundle.LoadVerified(path)
				if err != nil {
					t.Fatal(err)
				}
				last := loaded.Records[len(loaded.Records)-1]
				if last.Event.Data.(map[string]any)["bundle_hash"] != returnedHash {
					t.Fatal("written signature record does not contain the original-byte digest")
				}
				recordBytes, err := json.Marshal(last)
				if err != nil {
					t.Fatal(err)
				}
				separator := []byte(nil)
				if raw[len(raw)-1] != '\n' {
					separator = []byte{'\n'}
				}
				want := append(bytes.Clone(raw), separator...)
				physicalHash := sha256.Sum256(want)
				want = append(want, recordBytes...)
				want = append(want, '\n')
				if !bytes.Equal(signed, want) {
					t.Fatal("output must be original bytes + optional separator LF + signature JSON + LF")
				}
				report := verify.Verify(loaded, path, "")
				if !report.Integrity.ChainValid {
					t.Fatal("written bundle chain did not verify")
				}
				t.Logf("pass=%d input_len=%d input_tail=%x separator=%x record_offset=%d digest=%x physical_prefix_hash=%x chain_valid=%t signature=%+v",
					pass+1, len(raw), raw[max(0, len(raw)-4):], separator, len(raw)+len(separator), digest, physicalHash, report.Integrity.ChainValid, report.BundleSignature)
				if report.BundleSignature == nil || !report.BundleSignature.Verified {
					t.Errorf("pass %d: public verification failed: %+v", pass+1, report.BundleSignature)
				}
				if len(report.Signatures) != pass+1 {
					t.Fatalf("signature count=%d, want %d", len(report.Signatures), pass+1)
				}
				for _, sig := range report.Signatures {
					if !sig.Valid {
						t.Errorf("pass %d: signature failed after signing: %+v", pass+1, sig)
					}
				}
				raw = signed
			}
		})
	}
}
