package anchor

import (
	"bytes"
	"crypto/sha256"
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
)

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
