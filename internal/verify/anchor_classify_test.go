// SPDX-License-Identifier: MIT
package verify

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestClassifyAnchor_Absent(t *testing.T) {
	if got := ClassifyAnchor(newVerifyTestBundle(t), "bundle.atb", nil); got != AnchorAbsent {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorAbsent)
	}
}

func TestClassifyAnchor_PresentBadData(t *testing.T) {
	fixture := readAnchorTSRFixture(t)
	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeBundleAnchor, mustMarshalAnchorEventData(t, fixture), "2026-03-28T03:04:05Z")

	if got := ClassifyAnchor(b, "", nil); got != AnchorPresentBadData {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorPresentBadData)
	}
}

func TestClassifyAnchor_DigestMismatch(t *testing.T) {
	fixture := readAnchorTSRFixture(t)
	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeBundleAnchor, mustMarshalAnchorEventData(t, fixture), "2026-03-28T03:04:05Z")

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	if got := ClassifyAnchor(b, path, nil); got != AnchorPresentBadData {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorPresentBadData)
	}
}

func TestClassifyAnchor_Verified(t *testing.T) {
	fixture := readVerifiedAnchorTSRFixture(t)
	roots := verifiedAnchorFixtureRoots(t, fixture)

	b := buildVerifiedAnchorFixtureBundle(t)
	appendVerifyRecord(t, b, event.TypeBundleAnchor, mustMarshalAnchorEventData(t, fixture), "2026-03-28T04:05:06Z")

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	if got := ClassifyAnchor(b, path, roots); got != AnchorVerified {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorVerified)
	}
}

func TestClassifyAnchor_ChainNotVerified(t *testing.T) {
	fixture := readVerifiedAnchorTSRFixture(t)

	b := buildVerifiedAnchorFixtureBundle(t)
	appendVerifyRecord(t, b, event.TypeBundleAnchor, mustMarshalAnchorEventData(t, fixture), "2026-03-28T04:05:06Z")

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	if got := ClassifyAnchor(b, path, x509.NewCertPool()); got != AnchorDigestOnly {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorDigestOnly)
	}
}

func readAnchorTSRFixture(t testing.TB) []byte {
	t.Helper()

	fixture, err := os.ReadFile(filepath.Join("..", "anchor", "testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read anchor fixture: %v", err)
	}
	return fixture
}

func readVerifiedAnchorTSRFixture(t testing.TB) []byte {
	t.Helper()

	fixture, err := os.ReadFile(filepath.Join("testdata", "anchor_token_verified.tsr"))
	if err != nil {
		t.Fatalf("read verified anchor fixture: %v", err)
	}
	return fixture
}

func mustMarshalAnchorEventData(t testing.TB, tsrDER []byte) string {
	t.Helper()

	tsrHash := sha256.Sum256(tsrDER)
	payload := map[string]any{
		"bundle_hash":    "bundle-hash",
		"tsr_hash":       hex.EncodeToString(tsrHash[:]),
		"tsr_der":        base64.StdEncoding.EncodeToString(tsrDER),
		"certified_time": "2026-03-28T03:04:05Z",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal anchor payload: %v", err)
	}
	return string(raw)
}

func buildVerifiedAnchorFixtureBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := &bundle.Bundle{}
	if err := b.AppendWithOptions(event.TypeDevSession, "verified-anchor-fixture", &bundle.AppendOptions{
		Timestamp: "2026-03-28T03:04:05Z",
	}); err != nil {
		t.Fatalf("append verify fixture event: %v", err)
	}
	return b
}

func verifiedAnchorFixtureRoots(t testing.TB, tsrDER []byte) *x509.CertPool {
	t.Helper()

	certs := extractVerifiedAnchorFixtureCertificates(t, tsrDER)
	roots := x509.NewCertPool()
	for _, cert := range certs {
		roots.AddCert(cert)
	}
	return roots
}

func extractVerifiedAnchorFixtureCertificates(t testing.TB, tsrDER []byte) []*x509.Certificate {
	t.Helper()

	responseElems := parseFixtureSequence(t, tsrDER)
	if len(responseElems) < 2 {
		t.Fatalf("verified anchor fixture missing timestamp token")
	}

	contentInfoElems := parseFixtureSequence(t, responseElems[1].FullBytes)
	if len(contentInfoElems) < 2 {
		t.Fatalf("verified anchor fixture missing signed data")
	}

	var signedData struct {
		Version          int
		DigestAlgorithms asn1.RawValue
		EncapContentInfo asn1.RawValue
		Certificates     asn1.RawValue `asn1:"optional,tag:0"`
		CRLs             asn1.RawValue `asn1:"optional,tag:1"`
		SignerInfos      asn1.RawValue
	}
	if _, err := asn1.Unmarshal(contentInfoElems[1].Bytes, &signedData); err != nil {
		t.Fatalf("parse verified anchor signed data: %v", err)
	}
	if len(signedData.Certificates.Bytes) == 0 {
		t.Fatal("verified anchor fixture has no certificates")
	}

	certs := make([]*x509.Certificate, 0, 1)
	inner := signedData.Certificates.Bytes
	for len(inner) > 0 {
		var raw asn1.RawValue
		rest, err := asn1.Unmarshal(inner, &raw)
		if err != nil {
			t.Fatalf("parse verified anchor certificate set: %v", err)
		}
		cert, err := x509.ParseCertificate(raw.FullBytes)
		if err != nil {
			t.Fatalf("parse verified anchor certificate: %v", err)
		}
		certs = append(certs, cert)
		inner = rest
	}
	return certs
}

func parseFixtureSequence(t testing.TB, data []byte) []asn1.RawValue {
	t.Helper()

	var seq asn1.RawValue
	rest, err := asn1.Unmarshal(data, &seq)
	if err != nil {
		t.Fatalf("parse fixture sequence: %v", err)
	}
	if len(rest) != 0 {
		t.Fatalf("fixture sequence has %d trailing bytes", len(rest))
	}
	if seq.Class != asn1.ClassUniversal || seq.Tag != asn1.TagSequence || !seq.IsCompound {
		t.Fatalf("fixture value is not a sequence")
	}

	elems := make([]asn1.RawValue, 0, 8)
	inner := seq.Bytes
	for len(inner) > 0 {
		var elem asn1.RawValue
		remaining, err := asn1.Unmarshal(inner, &elem)
		if err != nil {
			t.Fatalf("parse fixture sequence element: %v", err)
		}
		elems = append(elems, elem)
		inner = remaining
	}
	return elems
}
