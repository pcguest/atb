// SPDX-License-Identifier: MIT
package acquisition

import (
	"errors"
	"fmt"

	"github.com/pcguest/atb/internal/event"
)

var (
	// ErrNoStableID indicates a source record has no stable identifier.
	ErrNoStableID = errors.New("acquisition: no stable source identity")

	// ErrInvalidDigest indicates a source digest is malformed.
	ErrInvalidDigest = errors.New("acquisition: invalid digest format")
)

// ReconciliationResult represents the outcome of comparing a new source record
// against existing acquired evidence.
type ReconciliationResult string

const (
	// ReconciliationNew indicates a genuinely new source record.
	ReconciliationNew ReconciliationResult = "new"

	// ReconciliationUnchanged indicates the source record is identical to a previously acquired record.
	ReconciliationUnchanged ReconciliationResult = "unchanged"

	// ReconciliationChanged indicates a source record with the same identity but different digest.
	ReconciliationChanged ReconciliationResult = "changed"

	// ReconciliationUnknown indicates the source record lacks a stable identity.
	ReconciliationUnknown ReconciliationResult = "unknown"
)

// ReconciliationOutcome represents the result of a reconciliation operation.
type ReconciliationOutcome struct {
	Result         ReconciliationResult
	PreviousDigest string
	CurrentDigest  string
	SourceIdentity event.SourceIdentity
	Finding        *FindingInfo
}

// FindingInfo represents a bounded finding derived from reconciliation.
type FindingInfo struct {
	Flag               string
	Severity           string
	Title              string
	Detail             string
	SourceRecordID     string
	PreviousDigest     string
	CurrentDigest      string
	AcquiredAt         string
	AcquiredAtPrevious string
}

// Reconciler handles reconciliation of incoming source records against
// previously acquired evidence.
type Reconciler struct {
	// KnownRecords maps source identity to the last known digest and acquisition metadata.
	KnownRecords map[string]*KnownRecord
}

// KnownRecord represents a previously acquired source record.
type KnownRecord struct {
	SourceIdentity event.SourceIdentity
	Digest         string
	AcquiredAt     string
	EventSequence  int
	Acquisition    *event.AcquisitionInfo
}

// NewReconciler creates a new reconciler with the given known records.
func NewReconciler(knownRecords map[string]*KnownRecord) *Reconciler {
	if knownRecords == nil {
		knownRecords = make(map[string]*KnownRecord)
	}
	return &Reconciler{KnownRecords: knownRecords}
}

// Reconcile compares a new source record against known records and returns
// the reconciliation outcome.
func (r *Reconciler) Reconcile(sourceIdentity event.SourceIdentity, currentDigest, currentAcquiredAt string, currentAcquisition *event.AcquisitionInfo) (*ReconciliationOutcome, error) {
	if sourceIdentity.System == "" || sourceIdentity.RecordID == "" {
		// No stable identity - treat as unknown
		return &ReconciliationOutcome{
			Result:         ReconciliationUnknown,
			CurrentDigest:  currentDigest,
			SourceIdentity: sourceIdentity,
		}, nil
	}

	key := r.identityKey(sourceIdentity)
	known, exists := r.KnownRecords[key]

	if !exists {
		// New record
		r.KnownRecords[key] = &KnownRecord{
			SourceIdentity: sourceIdentity,
			Digest:         currentDigest,
			AcquiredAt:     currentAcquiredAt,
			Acquisition:    currentAcquisition,
		}
		return &ReconciliationOutcome{
			Result:         ReconciliationNew,
			CurrentDigest:  currentDigest,
			SourceIdentity: sourceIdentity,
		}, nil
	}

	// Same identity - compare digests
	if known.Digest == currentDigest {
		return &ReconciliationOutcome{
			Result:         ReconciliationUnchanged,
			PreviousDigest: known.Digest,
			CurrentDigest:  currentDigest,
			SourceIdentity: sourceIdentity,
		}, nil
	}

	// Same identity, different digest - CHANGED
	finding := &FindingInfo{
		Flag:               "source_record_changed",
		Severity:           "high",
		Title:              "Source record representation changed",
		Detail:             fmt.Sprintf("Source record %s:%s was previously acquired with digest %s, now has digest %s", sourceIdentity.System, sourceIdentity.RecordID, known.Digest, currentDigest),
		SourceRecordID:     sourceIdentity.RecordID,
		PreviousDigest:     known.Digest,
		CurrentDigest:      currentDigest,
		AcquiredAt:         currentAcquiredAt,
		AcquiredAtPrevious: known.AcquiredAt,
	}

	// Update the known record with the new digest
	r.KnownRecords[key] = &KnownRecord{
		SourceIdentity: sourceIdentity,
		Digest:         currentDigest,
		AcquiredAt:     currentAcquiredAt,
		Acquisition:    currentAcquisition,
	}

	return &ReconciliationOutcome{
		Result:         ReconciliationChanged,
		PreviousDigest: known.Digest,
		CurrentDigest:  currentDigest,
		SourceIdentity: sourceIdentity,
		Finding:        finding,
	}, nil
}

// identityKey generates a stable key for a source identity.
func (r *Reconciler) identityKey(id event.SourceIdentity) string {
	return id.System + ":" + id.RecordID
}

// LoadFromBundle populates the reconciler's known records from an existing bundle.
func (r *Reconciler) LoadFromBundle(bundleRecords []interface{}) error {
	// This would iterate through bundle records and extract acquisition metadata
	// For now, this is a placeholder - implementation depends on bundle iteration API
	return nil
}

// GetKnownRecord returns the known record for a given source identity, if any.
func (r *Reconciler) GetKnownRecord(sourceIdentity event.SourceIdentity) *KnownRecord {
	key := r.identityKey(sourceIdentity)
	return r.KnownRecords[key]
}

// GetAllKnownRecords returns all known records.
func (r *Reconciler) GetAllKnownRecords() map[string]*KnownRecord {
	return r.KnownRecords
}
