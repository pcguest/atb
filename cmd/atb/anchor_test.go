package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
)

func TestRunAnchorWritesTSRAndAppendsAnchorEvent(t *testing.T) {
	bundlePath, originalHash := writeAnchorTestBundle(t)
	fixture, _, err := buildCLISignedAnchorFixture(originalHash)
	if err != nil {
		t.Fatalf("buildCLISignedAnchorFixture: %v", err)
	}
	stubTSATransport(t, fixture)

	result, err := runAnchor(anchorConfig{
		BundlePath: bundlePath,
		TSAURL:     "http://tsa.example.test",
	})
	if err != nil {
		t.Fatalf("runAnchor: %v", err)
	}
	t.Logf("anchor_event_data=%s", result.EventData)

	if result.TokenPath != bundlePath+".tsr" {
		t.Fatalf("unexpected token path: got %q", result.TokenPath)
	}

	tokenBytes, err := os.ReadFile(result.TokenPath)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if !bytes.Equal(tokenBytes, fixture) {
		t.Fatalf("unexpected token bytes")
	}

	loaded, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load anchored bundle: %v", err)
	}
	if got := loaded.Records[len(loaded.Records)-1].Event.Type; got != bundle.AnchorEventType {
		t.Fatalf("unexpected final event type: got %q", got)
	}

	anchorIndex, data, found, err := latestAnchorEventData(loaded)
	if err != nil {
		t.Fatalf("latestAnchorEventData: %v", err)
	}
	if !found {
		t.Fatalf("expected anchor event to be present")
	}
	if anchorIndex != len(loaded.Records)-1 {
		t.Fatalf("expected latest anchor event to be the final record")
	}

	if data.TSAURL != "http://tsa.example.test" {
		t.Fatalf("unexpected tsa_url: got %q", data.TSAURL)
	}
	if data.BundleHash != hex.EncodeToString(originalHash) {
		t.Fatalf("unexpected bundle_hash: got %q", data.BundleHash)
	}
	tsrHash := sha256.Sum256(fixture)
	if data.TSRHash != hex.EncodeToString(tsrHash[:]) {
		t.Fatalf("unexpected tsr_hash: got %q", data.TSRHash)
	}
	if data.TSRDER != base64.StdEncoding.EncodeToString(fixture) {
		t.Fatalf("unexpected tsr_der: got %q", data.TSRDER)
	}
	if data.CertifiedTime != "2026-03-28T03:04:05Z" {
		t.Fatalf("unexpected certified_time: got %q", data.CertifiedTime)
	}
}

func TestVerifyWithAnchorPassesOnAnchoredBundle(t *testing.T) {
	bundlePath, originalHash := writeAnchorTestBundle(t)
	fixture, roots, err := buildCLISignedAnchorFixture(originalHash)
	if err != nil {
		t.Fatalf("buildCLISignedAnchorFixture: %v", err)
	}
	stubTSATransport(t, fixture)
	verifyBundleAnchorRoots = roots
	t.Cleanup(func() {
		verifyBundleAnchorRoots = nil
	})

	if _, err := runAnchor(anchorConfig{
		BundlePath: bundlePath,
		TSAURL:     "http://tsa.example.test",
	}); err != nil {
		t.Fatalf("runAnchor: %v", err)
	}

	loaded, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load anchored bundle: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify anchored bundle chain: %v", err)
	}

	var out bytes.Buffer
	if err := verifyBundleAnchor(bundlePath, loaded, &out); err != nil {
		t.Fatalf("verifyBundleAnchor: %v", err)
	}
	if !strings.Contains(out.String(), "Anchor verified. Certified: 2026-03-28T03:04:05Z") {
		t.Fatalf("expected certified time output, got %q", out.String())
	}
}

func TestVerifyWithAnchorWarnsWhenTokenAbsent(t *testing.T) {
	bundlePath, originalHash := writeAnchorTestBundle(t)
	fixture, _, err := buildCLISignedAnchorFixture(originalHash)
	if err != nil {
		t.Fatalf("buildCLISignedAnchorFixture: %v", err)
	}
	stubTSATransport(t, fixture)

	result, err := runAnchor(anchorConfig{
		BundlePath: bundlePath,
		TSAURL:     "http://tsa.example.test",
	})
	if err != nil {
		t.Fatalf("runAnchor: %v", err)
	}
	if err := os.Remove(result.TokenPath); err != nil {
		t.Fatalf("remove token file: %v", err)
	}

	loaded, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load anchored bundle: %v", err)
	}

	var out bytes.Buffer
	if err := verifyBundleAnchor(bundlePath, loaded, &out); err != nil {
		t.Fatalf("verifyBundleAnchor: %v", err)
	}
	want := "No anchor token found at " + result.TokenPath + " — skipping anchor verification"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("expected missing token warning, got %q", out.String())
	}
}

func writeAnchorTestBundle(t *testing.T) (string, []byte) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "run.atb", bundle.BundleFile)
	b := bundle.New()
	appendTestBundleEvent(t, b, "dev.session", map[string]any{"ok": true})
	if err := b.Save(path); err != nil {
		t.Fatalf("save test bundle: %v", err)
	}
	hash, err := anchorpkg.HashBundle(path)
	if err != nil {
		t.Fatalf("hash test bundle: %v", err)
	}
	return path, hash
}

func stubTSATransport(t *testing.T, fixture []byte) {
	t.Helper()

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
			t.Fatalf("expected non-empty TSA request body")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     make(http.Header),
		}, nil
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

var (
	testOIDContentTypeSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	testOIDContentTypeTSTInfo    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
	testOIDDigestAlgorithmSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	testOIDSignatureAlgorithmRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
)

type testAlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type testMessageImprint struct {
	HashAlgorithm testAlgorithmIdentifier
	HashedMessage []byte
}

type testTSTInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint testMessageImprint
	SerialNumber   *big.Int
	GenTime        time.Time
}

type testIssuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

type testSignerInfo struct {
	Version            int
	SID                asn1.RawValue
	DigestAlgorithm    testAlgorithmIdentifier
	SignatureAlgorithm asn1.RawValue
	Signature          []byte
}

type testSignedData struct {
	Version          int
	DigestAlgorithms asn1.RawValue
	EncapContentInfo asn1.RawValue
	Certificates     asn1.RawValue `asn1:"optional,tag:0"`
	SignerInfos      asn1.RawValue
}

func buildCLISignedAnchorFixture(bundleHash []byte) ([]byte, *x509.CertPool, error) {
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
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "ATB CLI TSA Root"},
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
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "ATB CLI TSA"},
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

	tstInfoDER, err := asn1.Marshal(testTSTInfo{
		Version: 1,
		Policy:  asn1.ObjectIdentifier{1, 2, 3, 4},
		MessageImprint: testMessageImprint{
			HashAlgorithm: testAlgorithmIdentifier{
				Algorithm:  testOIDDigestAlgorithmSHA256,
				Parameters: testRawDER([]byte{0x05, 0x00}),
			},
			HashedMessage: bundleHash,
		},
		SerialNumber: big.NewInt(7),
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

	sidDER, err := asn1.Marshal(testIssuerAndSerialNumber{
		Issuer:       testRawDER(leafCert.RawIssuer),
		SerialNumber: leafCert.SerialNumber,
	})
	if err != nil {
		return nil, nil, err
	}
	signerInfoDER, err := asn1.Marshal(testSignerInfo{
		Version: 1,
		SID:     testRawDER(sidDER),
		DigestAlgorithm: testAlgorithmIdentifier{
			Algorithm:  testOIDDigestAlgorithmSHA256,
			Parameters: testRawDER([]byte{0x05, 0x00}),
		},
		SignatureAlgorithm: testRawDER(testMustMarshalAlgorithm(testOIDSignatureAlgorithmRSA)),
		Signature:          signature,
	})
	if err != nil {
		return nil, nil, err
	}

	encapContentDER, err := asn1.Marshal(struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue
	}{
		ContentType: testOIDContentTypeTSTInfo,
		Content:     testExplicitRaw(testMustMarshalOctetString(tstInfoDER)),
	})
	if err != nil {
		return nil, nil, err
	}

	signedDataDER, err := asn1.Marshal(testSignedData{
		Version:          1,
		DigestAlgorithms: testSetRaw(testMustMarshalAlgorithm(testOIDDigestAlgorithmSHA256)),
		EncapContentInfo: testRawDER(encapContentDER),
		Certificates: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      leafDER,
		},
		SignerInfos: testSetRaw(signerInfoDER),
	})
	if err != nil {
		return nil, nil, err
	}

	tokenDER, err := asn1.Marshal(struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue
	}{
		ContentType: testOIDContentTypeSignedData,
		Content:     testExplicitRaw(signedDataDER),
	})
	if err != nil {
		return nil, nil, err
	}

	responseDER, err := asn1.Marshal(struct {
		Status asn1.RawValue
		Token  asn1.RawValue
	}{
		Status: testRawDER(testMustMarshalStatus()),
		Token:  testRawDER(tokenDER),
	})
	if err != nil {
		return nil, nil, err
	}

	roots := x509.NewCertPool()
	roots.AddCert(rootCert)
	return responseDER, roots, nil
}

func testRawDER(der []byte) asn1.RawValue {
	return asn1.RawValue{FullBytes: der}
}

func testExplicitRaw(inner []byte) asn1.RawValue {
	return asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      inner,
	}
}

func testSetRaw(inner []byte) asn1.RawValue {
	return asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagSet,
		IsCompound: true,
		Bytes:      inner,
	}
}

func testMustMarshalStatus() []byte {
	der, err := asn1.Marshal(struct {
		Status int
	}{Status: 0})
	if err != nil {
		panic(err)
	}
	return der
}

func testMustMarshalOctetString(value []byte) []byte {
	der, err := asn1.Marshal(value)
	if err != nil {
		panic(err)
	}
	return der
}

func testMustMarshalAlgorithm(oid asn1.ObjectIdentifier) []byte {
	der, err := asn1.Marshal(testAlgorithmIdentifier{
		Algorithm:  oid,
		Parameters: testRawDER([]byte{0x05, 0x00}),
	})
	if err != nil {
		panic(err)
	}
	return der
}
