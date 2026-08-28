// SPDX-License-Identifier: MIT
package anchor

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildTSReqEncodesSHA256OID(t *testing.T) {
	req, err := buildTSReq(bytes.Repeat([]byte{0x42}, sha256.Size))
	if err != nil {
		t.Fatalf("buildTSReq: %v", err)
	}

	var raw asn1.RawValue
	rest, err := asn1.Unmarshal(req, &raw)
	if err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if len(rest) != 0 {
		t.Fatalf("expected no trailing bytes, got %d", len(rest))
	}

	sha256OIDDER := []byte{0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01}
	if !bytes.Contains(req, sha256OIDDER) {
		t.Fatalf("expected request to contain SHA-256 OID DER %x", sha256OIDDER)
	}
}

func TestHashBundle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	content := []byte("{\"event\":{\"seq\":0},\"hash\":\"abc\"}\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write temp bundle: %v", err)
	}

	got, err := HashBundle(path)
	if err != nil {
		t.Fatalf("HashBundle: %v", err)
	}

	want := sha256.Sum256(content)
	if !bytes.Equal(got, want[:]) {
		t.Fatalf("unexpected bundle hash: got %x want %x", got, want)
	}
}

func TestParseGenTime(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got, err := ParseGenTime(fixture)
	if err != nil {
		t.Fatalf("ParseGenTime: %v", err)
	}
	if got != "2026-03-28T03:04:05Z" {
		t.Fatalf("unexpected genTime: got %q", got)
	}
}

func TestRequestPostsTimestampQuery(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: got %s want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/timestamp-query" {
			t.Fatalf("unexpected content type: got %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if len(body) == 0 {
			t.Fatalf("expected non-empty request body")
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     make(http.Header),
		}, nil
	})

	got, err := Request("http://tsa.example.test", bytes.Repeat([]byte{0x24}, sha256.Size))
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if !bytes.Equal(got, fixture) {
		t.Fatalf("unexpected response bytes")
	}
}

func TestRequestRejectsOversizedResponse(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(io.LimitReader(zeroReader{}, maxTSAResponseBytes+1)),
			Header:     make(http.Header),
		}, nil
	})

	_, err := Request("https://tsa.example.test", bytes.Repeat([]byte{0x24}, sha256.Size))
	if err == nil || !strings.Contains(err.Error(), "TSA response exceeds") {
		t.Fatalf("expected oversized TSA response error, got %v", err)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestVerifyToken_DigestMismatch(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	err = VerifyToken(fixture, bytes.Repeat([]byte{0x01}, sha256.Size), x509.NewCertPool())
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %v", err)
	}
}

func TestVerifyToken_NoCerts(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	err = VerifyToken(fixture, make([]byte, sha256.Size), x509.NewCertPool())
	if err == nil || !strings.Contains(err.Error(), "no signer certificate") {
		t.Fatalf("expected no signer certificate error, got %v", err)
	}
}

func TestVerifyToken_DigestMatch_NoCertChain(t *testing.T) {
	bundleHash := sha256.Sum256([]byte("bundle-under-test"))
	token, _, err := buildSignedTestTSR(bundleHash[:], true)
	if err != nil {
		t.Fatalf("buildSignedTestTSR: %v", err)
	}

	err = VerifyToken(token, bundleHash[:], x509.NewCertPool())
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("expected certificate verification error, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func buildSignedTestTSR(bundleHash []byte, includeCerts bool) ([]byte, *x509.CertPool, error) {
	genTime := time.Date(2026, 3, 28, 3, 4, 5, 0, time.UTC)

	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	rootTemplate := &x509.Certificate{
		SerialNumber:          bigInt(1),
		Subject:               pkix.Name{CommonName: "ATB Test TSA Root"},
		NotBefore:             genTime.Add(-time.Hour),
		NotAfter:              genTime.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, err
	}
	rootCert, err := x509.ParseCertificate(rootDER)
	if err != nil {
		return nil, nil, err
	}

	leafTemplate := &x509.Certificate{
		SerialNumber:          bigInt(2),
		Subject:               pkix.Name{CommonName: "ATB Test TSA"},
		NotBefore:             genTime.Add(-time.Hour),
		NotAfter:              genTime.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
		BasicConstraintsValid: true,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, rootCert, &leafKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, err
	}
	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, nil, err
	}

	tstInfoDER, err := asn1.Marshal(tstInfo{
		Version: 1,
		Policy:  asn1.ObjectIdentifier{1, 2, 3, 4},
		MessageImprint: messageImprint{
			HashAlgorithm: algorithmIdentifier{
				Algorithm:  oidDigestAlgorithmSHA256,
				Parameters: rawDER([]byte{0x05, 0x00}),
			},
			HashedMessage: bundleHash,
		},
		SerialNumber: bigInt(7),
		GenTime:      genTime,
	})
	if err != nil {
		return nil, nil, err
	}

	digest := sha256.Sum256(tstInfoDER)
	signature, err := rsa.SignPKCS1v15(rand.Reader, leafKey, crypto.SHA256, digest[:])
	if err != nil {
		return nil, nil, err
	}

	sidDER, err := asn1.Marshal(issuerAndSerialNumber{
		Issuer:       rawDER(leafCert.RawIssuer),
		SerialNumber: leafCert.SerialNumber,
	})
	if err != nil {
		return nil, nil, err
	}
	signerInfoDER, err := asn1.Marshal(signerInfo{
		Version: 1,
		SID:     rawDER(sidDER),
		DigestAlgorithm: algorithmIdentifier{
			Algorithm:  oidDigestAlgorithmSHA256,
			Parameters: rawDER([]byte{0x05, 0x00}),
		},
		SignatureAlgorithm: rawDER(mustMarshalAlgorithm(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1})),
		Signature:          signature,
	})
	if err != nil {
		return nil, nil, err
	}

	encapContentDER, err := asn1.Marshal(struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue
	}{
		ContentType: oidContentTypeTSTInfo,
		Content:     explicitRaw(mustMarshalOctetString(tstInfoDER)),
	})
	if err != nil {
		return nil, nil, err
	}

	signed := signedData{
		Version:          1,
		DigestAlgorithms: setRaw(mustMarshalAlgorithm(oidDigestAlgorithmSHA256)),
		EncapContentInfo: rawDER(encapContentDER),
		SignerInfos:      setRaw(signerInfoDER),
	}
	if includeCerts {
		signed.Certificates = asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      leafDER,
		}
	}
	signedDataDER, err := asn1.Marshal(signed)
	if err != nil {
		return nil, nil, err
	}

	tokenDER, err := asn1.Marshal(struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue
	}{
		ContentType: oidContentTypeSignedData,
		Content:     explicitRaw(signedDataDER),
	})
	if err != nil {
		return nil, nil, err
	}

	responseDER, err := asn1.Marshal(struct {
		Status asn1.RawValue
		Token  asn1.RawValue
	}{
		Status: rawDER(mustMarshalStatus()),
		Token:  rawDER(tokenDER),
	})
	if err != nil {
		return nil, nil, err
	}

	roots := x509.NewCertPool()
	roots.AddCert(rootCert)
	return responseDER, roots, nil
}

func rawDER(der []byte) asn1.RawValue {
	return asn1.RawValue{FullBytes: der}
}

func explicitRaw(inner []byte) asn1.RawValue {
	return asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      inner,
	}
}

func setRaw(inner []byte) asn1.RawValue {
	return asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagSet,
		IsCompound: true,
		Bytes:      inner,
	}
}

func mustMarshalStatus() []byte {
	der, err := asn1.Marshal(struct {
		Status int
	}{Status: 0})
	if err != nil {
		panic(err)
	}
	return der
}

func mustMarshalOctetString(value []byte) []byte {
	der, err := asn1.Marshal(value)
	if err != nil {
		panic(err)
	}
	return der
}

func mustMarshalAlgorithm(oid asn1.ObjectIdentifier) []byte {
	der, err := asn1.Marshal(algorithmIdentifier{
		Algorithm:  oid,
		Parameters: rawDER([]byte{0x05, 0x00}),
	})
	if err != nil {
		panic(err)
	}
	return der
}

func bigInt(v int64) *big.Int {
	return big.NewInt(v)
}
