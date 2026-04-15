package verify

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

type AnchorVerifyResult int

type anchorStatus string

const (
	AnchorAbsent AnchorVerifyResult = iota
	AnchorPresentBadData
	AnchorDigestOnly
	AnchorVerified
)

const (
	anchorStatusVerified anchorStatus = "verified"
	anchorStatusPartial  anchorStatus = "partial"
	anchorStatusFailed   anchorStatus = "failed"

	anchorSummaryVerified = "anchor: verified (message imprint, signature, chain)"
	anchorSummaryPartial  = "anchor: partial (message imprint only - chain not verified)"
)

type anchorClassifyEventData struct {
	BundleHash string `json:"bundle_hash"`
	TSRDER     string `json:"tsr_der"`
}

type anchorInspection struct {
	Result                 AnchorVerifyResult
	Status                 anchorStatus
	Summary                string
	Reason                 string
	AnchorPresent          bool
	AnchorHash             string
	MessageImprintVerified bool
	SignatureVerified      bool
	CertChainVerified      bool
}

func ClassifyAnchor(b *bundle.Bundle, bundlePath string, roots *x509.CertPool) AnchorVerifyResult {
	return inspectAnchor(b, bundlePath, roots).Result
}

func inspectAnchor(b *bundle.Bundle, bundlePath string, roots *x509.CertPool) anchorInspection {
	if b == nil {
		return newFailedAnchorInspection(AnchorAbsent, false, "", "anchor record not present")
	}

	anchorIndex, payload, tokenBytes, found, err := latestAnchorTokenBytes(b)
	if !found {
		return newFailedAnchorInspection(AnchorAbsent, false, "", "anchor record not present")
	}
	if err != nil || len(tokenBytes) == 0 {
		return newFailedAnchorInspection(AnchorPresentBadData, true, payload.BundleHash, normaliseAnchorReason(err))
	}
	if strings.TrimSpace(bundlePath) == "" {
		return newFailedAnchorInspection(AnchorPresentBadData, true, payload.BundleHash, "bundle path required for anchor verification")
	}

	snapshotHash, err := hashBundleSnapshotBeforeAnchor(b, anchorIndex)
	if err != nil {
		return newFailedAnchorInspection(AnchorPresentBadData, true, payload.BundleHash, err.Error())
	}

	verification, err := anchorpkg.VerifyTokenDetailed(tokenBytes, snapshotHash, roots)
	if err != nil {
		inspection := anchorInspection{
			Result:                 AnchorPresentBadData,
			Status:                 anchorStatusFailed,
			Reason:                 normaliseAnchorReason(err),
			AnchorPresent:          true,
			AnchorHash:             payload.BundleHash,
			MessageImprintVerified: verification.MessageImprintVerified,
			SignatureVerified:      verification.SignatureVerified,
			CertChainVerified:      verification.CertChainVerified,
		}
		if isChainNotVerifiedAnchorError(err) {
			inspection.Result = AnchorDigestOnly
			inspection.Status = anchorStatusPartial
			inspection.Summary = anchorSummaryPartial
			return inspection
		}
		inspection.Summary = anchorFailedSummary(inspection.Reason)
		return inspection
	}

	return anchorInspection{
		Result:                 AnchorVerified,
		Status:                 anchorStatusVerified,
		Summary:                anchorSummaryVerified,
		AnchorPresent:          true,
		AnchorHash:             payload.BundleHash,
		Reason:                 "",
		MessageImprintVerified: verification.MessageImprintVerified,
		SignatureVerified:      verification.SignatureVerified,
		CertChainVerified:      verification.CertChainVerified,
	}
}

func xcScore(r AnchorVerifyResult) float64 {
	switch r {
	case AnchorPresentBadData:
		return 0.1
	case AnchorDigestOnly:
		return 0.5
	case AnchorVerified:
		return 1.0
	default:
		return 0.0
	}
}

func acScore(r AnchorVerifyResult) float64 {
	switch r {
	case AnchorVerified:
		return 1.0
	default:
		return 0.0
	}
}

func latestAnchorTokenBytes(b *bundle.Bundle) (int, anchorClassifyEventData, []byte, bool, error) {
	for i := len(b.Records) - 1; i >= 0; i-- {
		if b.Records[i].Event.Type != event.TypeBundleAnchor {
			continue
		}

		raw, ok := b.Records[i].Event.Data.(string)
		if !ok {
			return i, anchorClassifyEventData{}, nil, true, os.ErrInvalid
		}

		var payload anchorClassifyEventData
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return i, anchorClassifyEventData{}, nil, true, err
		}
		tokenBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.TSRDER))
		if err != nil {
			return i, payload, nil, true, err
		}
		return i, payload, tokenBytes, true, nil
	}
	return -1, anchorClassifyEventData{}, nil, false, nil
}

func hashBundleSnapshotBeforeAnchor(b *bundle.Bundle, anchorIndex int) ([]byte, error) {
	snapshot := &bundle.Bundle{
		Records: append([]bundle.Record(nil), b.Records[:anchorIndex]...),
	}

	tmp, err := os.CreateTemp("", "atb-verify-anchor-*.atb")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}
	defer os.Remove(tmpPath)

	if err := snapshot.Save(tmpPath); err != nil {
		return nil, err
	}
	return anchorpkg.HashBundle(tmpPath)
}

func isChainNotVerifiedAnchorError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "certificate verification failed")
}

func normaliseAnchorReason(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(err.Error()), "anchor: ")
}

func anchorFailedSummary(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "anchor: failed"
	}
	return "anchor: failed (" + reason + ")"
}

func newFailedAnchorInspection(result AnchorVerifyResult, present bool, anchorHash string, reason string) anchorInspection {
	return anchorInspection{
		Result:        result,
		Status:        anchorStatusFailed,
		Summary:       anchorFailedSummary(reason),
		Reason:        reason,
		AnchorPresent: present,
		AnchorHash:    anchorHash,
	}
}
