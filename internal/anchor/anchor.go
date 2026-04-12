package anchor

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"time"
)

const DefaultTSAURL = "http://timestamp.digicert.com"

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

type TokenVerification struct {
	MessageImprintVerified bool
	SignatureVerified      bool
	CertChainVerified      bool
	SignerCommonName       string
	IssuerCommonName       string
}

type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

// Request sends an RFC 3161 timestamp request for the given hash to tsaURL.
// Returns the raw DER-encoded TimeStampResponse bytes.
func Request(tsaURL string, hash []byte) ([]byte, error) {
	req, err := buildTSReq(hash)
	if err != nil {
		return nil, fmt.Errorf("anchor: build request: %w", err)
	}
	resp, err := http.Post(tsaURL, "application/timestamp-query", bytes.NewReader(req))
	if err != nil {
		return nil, fmt.Errorf("anchor: http post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anchor: TSA returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ParseGenTime extracts the genTime field from a DER-encoded TimeStampResponse.
// Returns RFC 3339 UTC string.
func ParseGenTime(tsrDER []byte) (string, error) {
	respElems, err := parseSequenceElements(tsrDER)
	if err != nil {
		return "", fmt.Errorf("anchor: parse response: %w", err)
	}
	if len(respElems) < 2 {
		return "", fmt.Errorf("anchor: parse response: timestamp token missing")
	}

	contentInfoElems, err := parseSequenceElements(respElems[1].FullBytes)
	if err != nil {
		return "", fmt.Errorf("anchor: parse content info: %w", err)
	}
	if len(contentInfoElems) < 2 {
		return "", fmt.Errorf("anchor: parse content info: missing signed data")
	}

	var contentType asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(contentInfoElems[0].FullBytes, &contentType); err != nil {
		return "", fmt.Errorf("anchor: parse content type: %w", err)
	}
	if !contentType.Equal(oidContentTypeSignedData) {
		return "", fmt.Errorf("anchor: parse content type: unexpected OID %v", contentType)
	}

	signedDataWrapper := contentInfoElems[1]
	if signedDataWrapper.Class != asn1.ClassContextSpecific || signedDataWrapper.Tag != 0 {
		return "", fmt.Errorf("anchor: parse signed data: expected explicit [0] wrapper")
	}

	signedDataElems, err := parseSequenceElements(signedDataWrapper.Bytes)
	if err != nil {
		return "", fmt.Errorf("anchor: parse signed data: %w", err)
	}
	if len(signedDataElems) < 3 {
		return "", fmt.Errorf("anchor: parse signed data: encapContentInfo missing")
	}

	encapContentElems, err := parseSequenceElements(signedDataElems[2].FullBytes)
	if err != nil {
		return "", fmt.Errorf("anchor: parse encap content: %w", err)
	}
	if len(encapContentElems) < 2 {
		return "", fmt.Errorf("anchor: parse encap content: eContent missing")
	}

	var eContentType asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(encapContentElems[0].FullBytes, &eContentType); err != nil {
		return "", fmt.Errorf("anchor: parse eContent type: %w", err)
	}
	if !eContentType.Equal(oidContentTypeTSTInfo) {
		return "", fmt.Errorf("anchor: parse eContent type: unexpected OID %v", eContentType)
	}

	eContentWrapper := encapContentElems[1]
	if eContentWrapper.Class != asn1.ClassContextSpecific || eContentWrapper.Tag != 0 {
		return "", fmt.Errorf("anchor: parse eContent: expected explicit [0] wrapper")
	}

	var tstInfoDER []byte
	if _, err := asn1.Unmarshal(eContentWrapper.Bytes, &tstInfoDER); err != nil {
		return "", fmt.Errorf("anchor: parse eContent octet string: %w", err)
	}

	tstInfoElems, err := parseSequenceElements(tstInfoDER)
	if err != nil {
		return "", fmt.Errorf("anchor: parse TSTInfo: %w", err)
	}
	if len(tstInfoElems) < 5 {
		return "", fmt.Errorf("anchor: parse TSTInfo: genTime missing")
	}

	var genTime time.Time
	if _, err := asn1.Unmarshal(tstInfoElems[4].FullBytes, &genTime); err != nil {
		return "", fmt.Errorf("anchor: parse genTime: %w", err)
	}
	return genTime.UTC().Format(time.RFC3339), nil
}

// HashBundle returns the SHA-256 hash of the canonical bundle content at path.
func HashBundle(path string) ([]byte, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- caller controls the bundle path
	if err != nil {
		return nil, fmt.Errorf("anchor: read bundle: %w", err)
	}
	sum := sha256.Sum256(raw)
	return sum[:], nil
}

func VerifyToken(tsrDER []byte, bundleHash []byte, roots *x509.CertPool) error {
	_, err := VerifyTokenDetailed(tsrDER, bundleHash, roots)
	return err
}

func VerifyTokenDetailed(tsrDER []byte, bundleHash []byte, roots *x509.CertPool) (TokenVerification, error) {
	result := TokenVerification{}

	parsed, err := parseToken(tsrDER)
	if err != nil {
		return result, err
	}

	if !bytes.Equal(parsed.Info.MessageImprint.HashedMessage, bundleHash) {
		return result, fmt.Errorf("anchor: digest mismatch")
	}
	result.MessageImprintVerified = true

	certs, err := extractCertificates(parsed.SignedDataDER)
	if err != nil {
		return result, err
	}
	if len(certs) == 0 {
		return result, fmt.Errorf("anchor: no signer certificate in timestamp token")
	}

	signers, err := extractSignerInfos(parsed.SignedData.SignerInfos.FullBytes)
	if err != nil {
		return result, err
	}
	if len(signers) == 0 {
		return result, fmt.Errorf("anchor: missing signer info")
	}

	signerCert, err := findSignerCertificate(signers[0], certs)
	if err != nil {
		return result, err
	}
	result.SignerCommonName = signerCert.Subject.CommonName
	result.IssuerCommonName = signerCert.Issuer.CommonName

	rootPool := roots
	if rootPool == nil {
		rootPool, err = x509.SystemCertPool()
		if err != nil {
			return result, fmt.Errorf("anchor: load system roots: %w", err)
		}
	}
	verifyOpts := x509.VerifyOptions{
		Roots:         rootPool,
		Intermediates: buildIntermediatesPool(certs, signerCert),
		CurrentTime:   parsed.Info.GenTime,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}
	if _, err := signerCert.Verify(verifyOpts); err != nil {
		return result, fmt.Errorf("anchor: certificate verification failed: %w", err)
	}
	result.CertChainVerified = true

	if err := verifySignerSignature(signers[0], signerCert, parsed.TSTInfoDER); err != nil {
		return result, fmt.Errorf("anchor: signature verification failed: %w", err)
	}
	result.SignatureVerified = true
	return result, nil
}

func buildIntermediatesPool(certs []*x509.Certificate, signer *x509.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	for _, cert := range certs {
		if !bytes.Equal(cert.Raw, signer.Raw) {
			pool.AddCert(cert)
		}
	}
	return pool
}

func extractCertificates(signedDataBytes []byte) ([]*x509.Certificate, error) {
	var parsed signedData
	if _, err := asn1.Unmarshal(signedDataBytes, &parsed); err != nil {
		return nil, fmt.Errorf("anchor: parse signed data: %w", err)
	}
	if len(parsed.Certificates.Bytes) == 0 {
		return []*x509.Certificate{}, nil
	}

	certs := make([]*x509.Certificate, 0, 4)
	inner := parsed.Certificates.Bytes
	for len(inner) > 0 {
		var raw asn1.RawValue
		rest, err := asn1.Unmarshal(inner, &raw)
		if err != nil {
			return nil, fmt.Errorf("anchor: parse certificate set: %w", err)
		}
		cert, err := x509.ParseCertificate(raw.FullBytes)
		if err != nil {
			return nil, fmt.Errorf("anchor: parse signer certificate: %w", err)
		}
		certs = append(certs, cert)
		inner = rest
	}
	return certs, nil
}

// buildTSReq constructs a minimal RFC 3161 TimeStampReq DER blob.
func buildTSReq(hash []byte) ([]byte, error) {
	// OID for SHA-256: 2.16.840.1.101.3.4.2.1
	sha256OID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	type pkix struct {
		Algorithm  asn1.ObjectIdentifier
		Parameters asn1.RawValue `asn1:"optional"`
	}
	type messageImprint struct {
		HashAlgorithm pkix
		HashedMessage []byte
	}
	type tsReq struct {
		Version        int
		MessageImprint messageImprint
		Nonce          *big.Int `asn1:"optional"`
		CertReq        bool     `asn1:"optional"`
	}
	req := tsReq{
		Version: 1,
		MessageImprint: messageImprint{
			HashAlgorithm: pkix{Algorithm: sha256OID},
			HashedMessage: hash,
		},
		CertReq: true,
	}
	return asn1.Marshal(req)
}

func parseSequenceElements(data []byte) ([]asn1.RawValue, error) {
	var seq asn1.RawValue
	rest, err := asn1.Unmarshal(data, &seq)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("unexpected trailing bytes")
	}
	if seq.Class != asn1.ClassUniversal || seq.Tag != asn1.TagSequence || !seq.IsCompound {
		return nil, fmt.Errorf("expected ASN.1 sequence")
	}

	elements := make([]asn1.RawValue, 0, 8)
	inner := seq.Bytes
	for len(inner) > 0 {
		var elem asn1.RawValue
		remaining, err := asn1.Unmarshal(inner, &elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		inner = remaining
	}
	return elements, nil
}

func parseSetElements(data []byte) ([]asn1.RawValue, error) {
	var set asn1.RawValue
	rest, err := asn1.Unmarshal(data, &set)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("unexpected trailing bytes")
	}
	if set.Class != asn1.ClassUniversal || set.Tag != asn1.TagSet || !set.IsCompound {
		return nil, fmt.Errorf("expected ASN.1 set")
	}

	elements := make([]asn1.RawValue, 0, 4)
	inner := set.Bytes
	for len(inner) > 0 {
		var elem asn1.RawValue
		remaining, err := asn1.Unmarshal(inner, &elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		inner = remaining
	}
	return elements, nil
}

type parsedToken struct {
	SignedDataDER []byte
	SignedData    signedData
	TSTInfoDER    []byte
	Info          tstInfo
}

func parseToken(tsrDER []byte) (parsedToken, error) {
	respElems, err := parseSequenceElements(tsrDER)
	if err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse response: %w", err)
	}
	if len(respElems) < 2 {
		return parsedToken{}, fmt.Errorf("anchor: parse response: timestamp token missing")
	}

	contentInfoElems, err := parseSequenceElements(respElems[1].FullBytes)
	if err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse content info: %w", err)
	}
	if len(contentInfoElems) < 2 {
		return parsedToken{}, fmt.Errorf("anchor: parse content info: missing signed data")
	}

	var contentType asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(contentInfoElems[0].FullBytes, &contentType); err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse content type: %w", err)
	}
	if !contentType.Equal(oidContentTypeSignedData) {
		return parsedToken{}, fmt.Errorf("anchor: parse content type: unexpected OID %v", contentType)
	}

	signedDataWrapper := contentInfoElems[1]
	if signedDataWrapper.Class != asn1.ClassContextSpecific || signedDataWrapper.Tag != 0 {
		return parsedToken{}, fmt.Errorf("anchor: parse signed data: expected explicit [0] wrapper")
	}

	var parsed signedData
	if _, err := asn1.Unmarshal(signedDataWrapper.Bytes, &parsed); err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse signed data: %w", err)
	}

	encapContentElems, err := parseSequenceElements(parsed.EncapContentInfo.FullBytes)
	if err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse encap content: %w", err)
	}
	if len(encapContentElems) < 2 {
		return parsedToken{}, fmt.Errorf("anchor: parse encap content: eContent missing")
	}

	var eContentType asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(encapContentElems[0].FullBytes, &eContentType); err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse eContent type: %w", err)
	}
	if !eContentType.Equal(oidContentTypeTSTInfo) {
		return parsedToken{}, fmt.Errorf("anchor: parse eContent type: unexpected OID %v", eContentType)
	}

	eContentWrapper := encapContentElems[1]
	if eContentWrapper.Class != asn1.ClassContextSpecific || eContentWrapper.Tag != 0 {
		return parsedToken{}, fmt.Errorf("anchor: parse eContent: expected explicit [0] wrapper")
	}

	var tstInfoDER []byte
	if _, err := asn1.Unmarshal(eContentWrapper.Bytes, &tstInfoDER); err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse eContent octet string: %w", err)
	}

	var info tstInfo
	if _, err := asn1.Unmarshal(tstInfoDER, &info); err != nil {
		return parsedToken{}, fmt.Errorf("anchor: parse TSTInfo: %w", err)
	}
	return parsedToken{
		SignedDataDER: signedDataWrapper.Bytes,
		SignedData:    parsed,
		TSTInfoDER:    tstInfoDER,
		Info:          info,
	}, nil
}

func extractSignerInfos(signerInfosDER []byte) ([]signerInfo, error) {
	rawInfos, err := parseSetElements(signerInfosDER)
	if err != nil {
		return nil, fmt.Errorf("anchor: parse signer infos: %w", err)
	}

	out := make([]signerInfo, 0, len(rawInfos))
	for _, raw := range rawInfos {
		var info signerInfo
		if _, err := asn1.Unmarshal(raw.FullBytes, &info); err != nil {
			return nil, fmt.Errorf("anchor: parse signer info: %w", err)
		}
		out = append(out, info)
	}
	return out, nil
}

func findSignerCertificate(info signerInfo, certs []*x509.Certificate) (*x509.Certificate, error) {
	if info.SID.Class == asn1.ClassContextSpecific && info.SID.Tag == 0 {
		for _, cert := range certs {
			if bytes.Equal(info.SID.Bytes, cert.SubjectKeyId) {
				return cert, nil
			}
		}
		return nil, fmt.Errorf("anchor: no signer certificate matches subject key identifier")
	}

	var sid issuerAndSerialNumber
	if _, err := asn1.Unmarshal(info.SID.FullBytes, &sid); err != nil {
		return nil, fmt.Errorf("anchor: parse signer identifier: %w", err)
	}
	for _, cert := range certs {
		if bytes.Equal(sid.Issuer.FullBytes, cert.RawIssuer) && sid.SerialNumber.Cmp(cert.SerialNumber) == 0 {
			return cert, nil
		}
	}
	return nil, fmt.Errorf("anchor: no signer certificate matches signer info")
}

func verifySignerSignature(info signerInfo, cert *x509.Certificate, content []byte) error {
	if !info.DigestAlgorithm.Algorithm.Equal(oidDigestAlgorithmSHA256) {
		return fmt.Errorf("unsupported digest algorithm %v", info.DigestAlgorithm.Algorithm)
	}

	sum := sha256.Sum256(content)
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], info.Signature); err != nil {
			return err
		}
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pub, sum[:], info.Signature) {
			return fmt.Errorf("ECDSA signature invalid")
		}
	default:
		return fmt.Errorf("unsupported public key algorithm %v", cert.PublicKeyAlgorithm)
	}
	return nil
}
