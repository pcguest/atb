// SPDX-License-Identifier: MIT
package verify

import "github.com/pcguest/atb/internal/bundle"

// SubScoresForBundle derives profile sub-scores using the same anchor-aware
// path as bundle verification.
func SubScoresForBundle(b *bundle.Bundle, bundlePath string, profileID string) map[string]float64 {
	return subScoresForBundleWithAnchorResult(b, profileID, ClassifyAnchor(b, bundlePath, nil))
}

func subScoresForBundleWithAnchorResult(b *bundle.Bundle, profileID string, anchorResult AnchorVerifyResult) map[string]float64 {
	if b == nil {
		return zeroSubScores()
	}

	profile := ProfileByID(profileID)
	if profile == nil {
		return zeroSubScores()
	}

	return subScoresForProfile(profile, b.Records, anchorResult)
}

func zeroSubScores() map[string]float64 {
	return map[string]float64{
		"EC": 0,
		"FC": 0,
		"RC": 0,
		"TC": 0,
		"SC": 0,
		"XC": 0,
		"AC": 0,
		"GC": 0,
	}
}
