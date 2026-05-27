// SPDX-License-Identifier: MIT
// Package signing defines Custos policy and result types for ATB automatic
// bundle signing.
package signing

import (
	"time"

	atbv1 "github.com/pcguest/atb/pkg/api/v1"
)

// KeySource identifies where the signing key is loaded from.
type KeySource int

const (
	KeySourceLocalFile KeySource = iota // local Ed25519 key file path
	KeySourceKMS                        // KMS reference string
)

// SigningPolicy configures automatic bundle signing for an organisation.
// ATB core signs each bundle event automatically via
// internal/signer.NewLocalSigner; this policy determines which key source and
// rotation schedule apply per org.
type SigningPolicy struct {
	OrgID            string
	KeySource        KeySource
	KeyRef           string // file path or KMS key ID
	RotationSchedule string // cron expression, e.g. "0 0 * * 0"
	RequireTSA       bool   // require RFC 3161 timestamp anchoring
	CreatedAt        time.Time
}

// PolicyStore persists and retrieves signing policies per organisation.
type PolicyStore interface {
	// TODO: define persistence and lookup semantics.
	Save(policy SigningPolicy) error
	// TODO: define persistence and lookup semantics.
	Get(orgID string) (*SigningPolicy, error)
}

// BundleSigningResult records the automatic signing outcome for a bundle.
type BundleSigningResult struct {
	BundleID    string
	OrgID       string
	Signed      bool
	TSAAnchored bool
	SignedAt    time.Time
	KeyRef      string
	// NB: Signed is set by ATB core automatically at capture time.
	// Custos records the outcome; it does not trigger signing.
	EventRecord *atbv1.EventRecordDTO
}
