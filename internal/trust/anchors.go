package trust

import (
	"github.com/pcguest/atb/internal/anchorverify"
	"github.com/pcguest/atb/internal/bundle"
)

type AnchorVerificationResult = anchorverify.AnchorVerificationResult

// VerifyAnchors scans the bundle for anchor records and verifies each one.
func VerifyAnchors(b *bundle.Bundle) []AnchorVerificationResult {
	return anchorverify.VerifyAnchors(b)
}
