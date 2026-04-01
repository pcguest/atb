//go:build generate

package testdata

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestGenerateAnchorTokenVerifiedFixture(t *testing.T) {
	bundleBytes := buildVerifiedAnchorFixtureBundleBytes(t)
	bundleHash := sha256.Sum256(bundleBytes)

	tokenDER, roots, err := buildVerifiedAnchorFixtureToken(bundleHash[:])
	if err != nil {
		t.Fatalf("build verified anchor fixture token: %v", err)
	}
	if err := anchorpkg.VerifyToken(tokenDER, bundleHash[:], roots); err != nil {
		t.Fatalf("verify generated anchor fixture token: %v", err)
	}

	path := filepath.Join(generatorDir(t), "anchor_token_verified.tsr")
	if err := os.WriteFile(path, tokenDER, 0600); err != nil {
		t.Fatalf("write anchor token fixture: %v", err)
	}
}

func buildVerifiedAnchorFixtureBundleBytes(t *testing.T) []byte {
	t.Helper()

	b := &bundle.Bundle{}
	if err := b.AppendWithOptions(event.TypeDevSession, "verified-anchor-fixture", &bundle.AppendOptions{
		Timestamp: "2026-03-28T03:04:05Z",
	}); err != nil {
		t.Fatalf("append verified anchor fixture event: %v", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, record := range b.Records {
		if err := enc.Encode(record); err != nil {
			t.Fatalf("encode verified anchor fixture bundle: %v", err)
		}
	}
	return buf.Bytes()
}

func buildVerifiedAnchorFixtureToken(bundleHash []byte) ([]byte, *x509.CertPool, error) {
	genTime := time.Date(2026, 3, 28, 4, 5, 6, 0, time.UTC)
	rng := newDeterministicReader("atb-anchor-token-verified-fixture-v1")

	key, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          bigInt(1),
		Subject:               pkix.Name{CommonName: "ATB Verified Anchor Fixture TSA"},
		NotBefore:             genTime.Add(-time.Hour),
		NotAfter:              genTime.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	certDER, err := x509.CreateCertificate(rng, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(certDER)
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
	signature, err := rsa.SignPKCS1v15(rng, key, crypto.SHA256, digest[:])
	if err != nil {
		return nil, nil, err
	}

	sidDER, err := asn1.Marshal(issuerAndSerialNumber{
		Issuer:       rawDER(cert.RawIssuer),
		SerialNumber: cert.SerialNumber,
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

	signedDataDER, err := asn1.Marshal(signedData{
		Version:          1,
		DigestAlgorithms: setRaw(mustMarshalAlgorithm(oidDigestAlgorithmSHA256)),
		EncapContentInfo: rawDER(encapContentDER),
		Certificates: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      certDER,
		},
		SignerInfos: setRaw(signerInfoDER),
	})
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
	roots.AddCert(cert)
	return responseDER, roots, nil
}

func generatorDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve generator path")
	}
	return filepath.Dir(file)
}

type deterministicReader struct {
	seed    []byte
	counter uint64
	buffer  []byte
}

func newDeterministicReader(seed string) *deterministicReader {
	return &deterministicReader{seed: []byte(seed)}
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(r.buffer) == 0 {
			block := make([]byte, len(r.seed)+8)
			copy(block, r.seed)
			putUint64(block[len(r.seed):], r.counter)
			sum := sha256.Sum256(block)
			r.buffer = append(r.buffer[:0], sum[:]...)
			r.counter++
		}
		copied := copy(p[n:], r.buffer)
		r.buffer = r.buffer[copied:]
		n += copied
	}
	return n, nil
}

func putUint64(dst []byte, value uint64) {
	for i := 7; i >= 0; i-- {
		dst[i] = byte(value)
		value >>= 8
	}
}

var (
	oidContentTypeSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidContentTypeTSTInfo    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
	oidDigestAlgorithmSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
)

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type signedData struct {
	Version          int
	DigestAlgorithms asn1.RawValue
	EncapContentInfo asn1.RawValue
	Certificates     asn1.RawValue `asn1:"optional,tag:0"`
	CRLs             asn1.RawValue `asn1:"optional,tag:1"`
	SignerInfos      asn1.RawValue
}

type signerInfo struct {
	Version            int
	SID                asn1.RawValue
	DigestAlgorithm    algorithmIdentifier
	SignedAttrs        asn1.RawValue `asn1:"optional,tag:0"`
	SignatureAlgorithm asn1.RawValue
	Signature          []byte
	UnsignedAttrs      asn1.RawValue `asn1:"optional,tag:1"`
}

type tstInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint messageImprint
	SerialNumber   *big.Int
	GenTime        time.Time
}

type messageImprint struct {
	HashAlgorithm algorithmIdentifier
	HashedMessage []byte
}

type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
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
