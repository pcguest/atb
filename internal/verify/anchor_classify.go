package verify

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

type AnchorVerifyResult int

const (
	AnchorAbsent AnchorVerifyResult = iota
	AnchorPresentBadData
	AnchorDigestOnly
	AnchorVerified
)

type anchorClassifyEventData struct {
	TSRDER string `json:"tsr_der"`
}

func ClassifyAnchor(b *bundle.Bundle, bundlePath string) AnchorVerifyResult {
	if b == nil {
		return AnchorAbsent
	}

	anchorIndex, tokenBytes, found, err := latestAnchorTokenBytes(b)
	if !found {
		return AnchorAbsent
	}
	if err != nil || len(tokenBytes) == 0 {
		return AnchorPresentBadData
	}
	if strings.TrimSpace(bundlePath) == "" {
		return AnchorPresentBadData
	}

	snapshotHash, err := hashBundleSnapshotBeforeAnchor(b, anchorIndex)
	if err != nil {
		return AnchorPresentBadData
	}

	if err := anchorpkg.VerifyToken(tokenBytes, snapshotHash, nil); err != nil {
		if isDigestOnlyAnchorError(err) {
			return AnchorDigestOnly
		}
		return AnchorPresentBadData
	}
	return AnchorVerified
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
	case AnchorDigestOnly:
		return 0.4
	case AnchorVerified:
		return 1.0
	default:
		return 0.0
	}
}

func latestAnchorTokenBytes(b *bundle.Bundle) (int, []byte, bool, error) {
	for i := len(b.Records) - 1; i >= 0; i-- {
		if b.Records[i].Event.Type != event.TypeBundleAnchor {
			continue
		}

		raw, ok := b.Records[i].Event.Data.(string)
		if !ok {
			return i, nil, true, os.ErrInvalid
		}

		var payload anchorClassifyEventData
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return i, nil, true, err
		}
		tokenBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.TSRDER))
		if err != nil {
			return i, nil, true, err
		}
		return i, tokenBytes, true, nil
	}
	return -1, nil, false, nil
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

func isDigestOnlyAnchorError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "certificate verification failed") || strings.Contains(msg, "signature verification failed")
}
