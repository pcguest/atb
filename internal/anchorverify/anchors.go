package anchorverify

import (
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

var (
	oidSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidTSTInfo    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
	oidSHA256     = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
)

type trustAnchorEventData struct {
	TSAURL        string `json:"tsa_url"`
	BundleHash    string `json:"bundle_hash"`
	TSRHash       string `json:"tsr_hash"`
	TSRDER        string `json:"tsr_der"`
	CertifiedTime string `json:"certified_time"`
}

type parsedTimeStampResponse struct {
	status        int
	hashAlgorithm asn1.ObjectIdentifier
	hashedMessage []byte
	genTime       time.Time
}

// AnchorVerificationResult describes the outcome of verifying one RFC 3161
// anchor record in a bundle.
type AnchorVerificationResult struct {
	AnchorIndex         int    `json:"anchor_index"`
	BundleDigest        string `json:"bundle_digest"`
	TSAGenTime          string `json:"tsa_gen_time"`
	MessageImprintMatch bool   `json:"message_imprint_match"`
	// TSAVerified is true only when MessageImprintMatch is true and the
	// PKIStatus in the TimeStampResponse is GRANTED (0).
	// Certificate chain verification is deferred to v2.
	TSAVerified bool   `json:"tsa_verified"`
	Error       string `json:"error,omitempty"`
}

// VerifyAnchors scans the bundle for anchor records and verifies each one.
// Returns one result per anchor record and an empty slice when none exist.
func VerifyAnchors(b *bundle.Bundle) []AnchorVerificationResult {
	if b == nil {
		return nil
	}

	results := make([]AnchorVerificationResult, 0, len(b.Records))
	for i, record := range b.Records {
		if record.Event.Type != event.TypeBundleAnchor {
			continue
		}

		result := AnchorVerificationResult{AnchorIndex: i}
		data, err := parseTrustAnchorEventData(record.Event.Data)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		result.BundleDigest = data.BundleHash
		result.TSAGenTime = data.CertifiedTime

		tsrDER, err := decodeEmbeddedTSRBytes(data.TSRDER, data.TSRHash)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		parsed, err := parseTimeStampResponse(tsrDER)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		result.TSAGenTime = parsed.genTime.UTC().Format(time.RFC3339)
		messageImprintMatch, err := messageImprintMatches(data.BundleHash, parsed.hashAlgorithm, parsed.hashedMessage)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		result.MessageImprintMatch = messageImprintMatch
		result.TSAVerified = messageImprintMatch && parsed.status == 0
		results = append(results, result)
	}

	return results
}

func parseTrustAnchorEventData(data any) (trustAnchorEventData, error) {
	raw, ok := data.(string)
	if !ok {
		return trustAnchorEventData{}, fmt.Errorf("anchor event data must be a JSON string")
	}

	var parsed trustAnchorEventData
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return trustAnchorEventData{}, fmt.Errorf("parse anchor event data: %w", err)
	}
	return parsed, nil
}

func decodeEmbeddedTSRBytes(tsrDER string, tsrHash string) ([]byte, error) {
	if value := strings.TrimSpace(tsrDER); value != "" {
		return decodeBase64TSRBytes(value)
	}

	value := strings.TrimSpace(tsrHash)
	switch {
	case value == "":
		return nil, fmt.Errorf("anchor event tsr_der field is empty; embedded TimeStampResponse bytes are unavailable")
	case isHexDigest(value):
		return nil, fmt.Errorf("anchor event tsr_hash field contains only the token digest; embedded TimeStampResponse bytes are unavailable")
	default:
		return decodeBase64TSRBytes(value)
	}
}

func decodeBase64TSRBytes(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode embedded TimeStampResponse bytes: %w", err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("embedded TimeStampResponse bytes are empty")
	}
	return decoded, nil
}

func isHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func parseTimeStampResponse(tsrDER []byte) (parsedTimeStampResponse, error) {
	responseElems, err := parseSequenceElements(tsrDER)
	if err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse TimeStampResponse: %w", err)
	}
	if len(responseElems) < 1 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse TimeStampResponse: PKIStatusInfo missing")
	}

	status, err := parsePKIStatus(responseElems[0].FullBytes)
	if err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse PKIStatusInfo: %w", err)
	}
	if len(responseElems) < 2 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse TimeStampResponse: timestamp token missing")
	}

	contentInfoElems, err := parseSequenceElements(responseElems[1].FullBytes)
	if err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse ContentInfo: %w", err)
	}
	if len(contentInfoElems) < 2 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse ContentInfo: signed data missing")
	}

	var contentType asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(contentInfoElems[0].FullBytes, &contentType); err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse ContentInfo type: %w", err)
	}
	if !contentType.Equal(oidSignedData) {
		return parsedTimeStampResponse{}, fmt.Errorf("parse ContentInfo type: unexpected OID %v", contentType)
	}

	signedDataWrapper := contentInfoElems[1]
	if signedDataWrapper.Class != asn1.ClassContextSpecific || signedDataWrapper.Tag != 0 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse SignedData: expected explicit [0] wrapper")
	}

	signedDataElems, err := parseSequenceElements(signedDataWrapper.Bytes)
	if err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse SignedData: %w", err)
	}
	if len(signedDataElems) < 3 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse SignedData: encapContentInfo missing")
	}

	encapContentElems, err := parseSequenceElements(signedDataElems[2].FullBytes)
	if err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse EncapsulatedContentInfo: %w", err)
	}
	if len(encapContentElems) < 2 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse EncapsulatedContentInfo: eContent missing")
	}

	var eContentType asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(encapContentElems[0].FullBytes, &eContentType); err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse eContent type: %w", err)
	}
	if !eContentType.Equal(oidTSTInfo) {
		return parsedTimeStampResponse{}, fmt.Errorf("parse eContent type: unexpected OID %v", eContentType)
	}

	eContentWrapper := encapContentElems[1]
	if eContentWrapper.Class != asn1.ClassContextSpecific || eContentWrapper.Tag != 0 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse eContent: expected explicit [0] wrapper")
	}

	var tstInfoDER []byte
	if _, err := asn1.Unmarshal(eContentWrapper.Bytes, &tstInfoDER); err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse TSTInfo OCTET STRING: %w", err)
	}

	tstInfoElems, err := parseSequenceElements(tstInfoDER)
	if err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse TSTInfo: %w", err)
	}
	if len(tstInfoElems) < 5 {
		return parsedTimeStampResponse{}, fmt.Errorf("parse TSTInfo: missing required fields")
	}

	hashAlgorithm, hashedMessage, err := parseMessageImprint(tstInfoElems[2].FullBytes)
	if err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse message imprint: %w", err)
	}

	var genTime time.Time
	if _, err := asn1.Unmarshal(tstInfoElems[4].FullBytes, &genTime); err != nil {
		return parsedTimeStampResponse{}, fmt.Errorf("parse genTime: %w", err)
	}

	return parsedTimeStampResponse{
		status:        status,
		hashAlgorithm: hashAlgorithm,
		hashedMessage: hashedMessage,
		genTime:       genTime,
	}, nil
}

func parsePKIStatus(der []byte) (int, error) {
	elems, err := parseSequenceElements(der)
	if err != nil {
		return 0, err
	}
	if len(elems) == 0 {
		return 0, fmt.Errorf("PKIStatus missing")
	}

	var status int
	if _, err := asn1.Unmarshal(elems[0].FullBytes, &status); err != nil {
		return 0, err
	}
	return status, nil
}

func parseMessageImprint(der []byte) (asn1.ObjectIdentifier, []byte, error) {
	elems, err := parseSequenceElements(der)
	if err != nil {
		return nil, nil, err
	}
	if len(elems) != 2 {
		return nil, nil, fmt.Errorf("message imprint should have algorithm and digest")
	}

	algorithmElems, err := parseSequenceElements(elems[0].FullBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse algorithm identifier: %w", err)
	}
	if len(algorithmElems) == 0 {
		return nil, nil, fmt.Errorf("algorithm identifier missing algorithm OID")
	}

	var algorithm asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(algorithmElems[0].FullBytes, &algorithm); err != nil {
		return nil, nil, fmt.Errorf("parse algorithm OID: %w", err)
	}
	if !algorithm.Equal(oidSHA256) {
		return nil, nil, fmt.Errorf("unsupported message imprint algorithm %v", algorithm)
	}

	var digest []byte
	if _, err := asn1.Unmarshal(elems[1].FullBytes, &digest); err != nil {
		return nil, nil, fmt.Errorf("parse message imprint digest: %w", err)
	}
	return algorithm, digest, nil
}

func messageImprintMatches(bundleHash string, algorithm asn1.ObjectIdentifier, hashedMessage []byte) (bool, error) {
	if !algorithm.Equal(oidSHA256) {
		return false, fmt.Errorf("unsupported digest algorithm %v", algorithm)
	}

	expected, err := hex.DecodeString(strings.TrimSpace(bundleHash))
	if err != nil {
		return false, fmt.Errorf("decode bundle hash: %w", err)
	}
	return string(expected) == string(hashedMessage), nil
}

func parseSequenceElements(der []byte) ([]asn1.RawValue, error) {
	var sequence asn1.RawValue
	if _, err := asn1.Unmarshal(der, &sequence); err != nil {
		return nil, err
	}
	if sequence.Tag != asn1.TagSequence || !sequence.IsCompound {
		return nil, fmt.Errorf("expected ASN.1 SEQUENCE")
	}

	data := sequence.Bytes
	elems := make([]asn1.RawValue, 0, 8)
	for len(data) > 0 {
		var elem asn1.RawValue
		rest, err := asn1.Unmarshal(data, &elem)
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
		data = rest
	}
	return elems, nil
}
