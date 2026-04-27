// SPDX-License-Identifier: MIT
package verify

import (
	"errors"
	"math"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// CorroborationPolicy controls how verified external evidence adjusts the base
// CAS score. A nil policy disables corroboration bonuses and is
// backward-compatible with the v1.8.0 output.
type CorroborationPolicy struct {
	// AnchorBonus is added when a verified RFC 3161 TSA anchor is present.
	AnchorBonus float64 `json:"anchor_bonus"`
	// SignatureBonus is added when a valid Ed25519 bundle signature is present.
	SignatureBonus float64 `json:"signature_bonus"`
	// SnapshotBonus is added when at least one verified atb.snapshot event is present.
	SnapshotBonus float64 `json:"snapshot_bonus"`
	// MaxBonus caps the total bonus regardless of combination. Must be ≤ 0.15.
	MaxBonus float64 `json:"max_bonus"`
}

// DefaultCorroborationPolicy returns the standard policy used by atb verify
// when --with-anchor is present.
func DefaultCorroborationPolicy() *CorroborationPolicy {
	return &CorroborationPolicy{
		AnchorBonus:    0.05,
		SignatureBonus: 0.03,
		SnapshotBonus:  0.02,
		MaxBonus:       0.10,
	}
}

// Validate returns an error if the policy's component bonuses sum exceeds MaxBonus.
func (p *CorroborationPolicy) Validate() error {
	if p == nil {
		return nil
	}
	sum := p.AnchorBonus + p.SignatureBonus + p.SnapshotBonus
	if sum > p.MaxBonus {
		return errors.New("corroboration policy: component bonuses sum exceeds MaxBonus")
	}
	return nil
}

// computeCorroborationBonus returns the capped bonus for the given evidence
// combination. Returns 0 when p is nil.
func computeCorroborationBonus(p *CorroborationPolicy, anchorVerified, signatureVerified, snapshotVerified bool) float64 {
	if p == nil {
		return 0.0
	}
	bonus := 0.0
	if anchorVerified {
		bonus += p.AnchorBonus
	}
	if signatureVerified {
		bonus += p.SignatureBonus
	}
	if snapshotVerified {
		bonus += p.SnapshotBonus
	}
	return math.Min(bonus, p.MaxBonus)
}

// hasVerifiedSnapshot reports whether the bundle contains at least one
// atb.snapshot event whose bundle_hash is consistent with the bundle prefix.
func hasVerifiedSnapshot(records []bundle.Record) bool {
	hasSnapshot := false
	for _, r := range records {
		if r.Event.Type == event.TypeSnapshot {
			hasSnapshot = true
			break
		}
	}
	if !hasSnapshot {
		return false
	}
	return len(VerifySnapshotHashes(records)) == 0
}
