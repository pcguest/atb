package trust

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
		return parsedTimeStampResponse{}, fmt.Errorf("parse eContent octet string: %w", err)
	}

	hashAlgorithm, hashedMessage, genTime, err := parseTSTInfo(tstInfoDER)
	if err != nil {
		return parsedTimeStampResponse{}, err
	}

	return parsedTimeStampResponse{
		status:        status,
		hashAlgorithm: hashAlgorithm,
		hashedMessage: hashedMessage,
		genTime:       genTime,
	}, nil
}

func parsePKIStatus(statusInfoDER []byte) (int, error) {
	statusElems, err := parseSequenceElements(statusInfoDER)
	if err != nil {
		return 0, err
	}
	if len(statusElems) == 0 {
		return 0, fmt.Errorf("status integer missing")
	}

	var status int
	if _, err := asn1.Unmarshal(statusElems[0].FullBytes, &status); err != nil {
		return 0, err
	}
	return status, nil
}

func parseTSTInfo(tstInfoDER []byte) (asn1.ObjectIdentifier, []byte, time.Time, error) {
	tstInfoElems, err := parseSequenceElements(tstInfoDER)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("parse TSTInfo: %w", err)
	}
	if len(tstInfoElems) < 5 {
		return nil, nil, time.Time{}, fmt.Errorf("parse TSTInfo: required fields missing")
	}

	hashAlgorithm, hashedMessage, err := parseMessageImprint(tstInfoElems[2].FullBytes)
	if err != nil {
		return nil, nil, time.Time{}, err
	}

	var genTime time.Time
	if _, err := asn1.Unmarshal(tstInfoElems[4].FullBytes, &genTime); err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("parse TSTInfo genTime: %w", err)
	}

	return hashAlgorithm, hashedMessage, genTime, nil
}

func parseMessageImprint(messageImprintDER []byte) (asn1.ObjectIdentifier, []byte, error) {
	messageImprintElems, err := parseSequenceElements(messageImprintDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parse messageImprint: %w", err)
	}
	if len(messageImprintElems) < 2 {
		return nil, nil, fmt.Errorf("parse messageImprint: required fields missing")
	}

	algorithmElems, err := parseSequenceElements(messageImprintElems[0].FullBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse messageImprint algorithm: %w", err)
	}
	if len(algorithmElems) == 0 {
		return nil, nil, fmt.Errorf("parse messageImprint algorithm: OID missing")
	}

	var oid asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(algorithmElems[0].FullBytes, &oid); err != nil {
		return nil, nil, fmt.Errorf("parse messageImprint algorithm OID: %w", err)
	}

	var hashedMessage []byte
	if _, err := asn1.Unmarshal(messageImprintElems[1].FullBytes, &hashedMessage); err != nil {
		return nil, nil, fmt.Errorf("parse messageImprint digest: %w", err)
	}
	return oid, hashedMessage, nil
}

// NOTE: certificate chain verification is deferred to v2. In v1, only
// the PKIStatus and messageImprint are checked. Callers must not treat
// TSAVerified=true as proof of TSA identity without chain verification.
func messageImprintMatches(bundleDigest string, algorithmOID asn1.ObjectIdentifier, hashedMessage []byte) (bool, error) {
	if !algorithmOID.Equal(oidSHA256) {
		return false, nil
	}

	digestBytes, err := hex.DecodeString(bundleDigest)
	if err != nil {
		return false, fmt.Errorf("decode bundle digest: %w", err)
	}
	return string(digestBytes) == string(hashedMessage), nil
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
