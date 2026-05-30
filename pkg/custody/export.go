// SPDX-License-Identifier: MIT
package custody

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
)

const (
	// BundleExportVersion identifies the ATB-to-custody handoff wire object.
	BundleExportVersion = "atb.custody.bundle_export.v1"
	receiptIDPrefix     = "sha256-"
)

// ErrInvalidExport is returned when a BundleExport cannot be serialised.
var ErrInvalidExport = errors.New("custody: invalid bundle export")

// WireExporter serialises a custody handoff object to its wire representation.
type WireExporter interface {
	Export() ([]byte, error)
}

var _ WireExporter = BundleExport{}

// ExportOptions configures local bundle export for custody ingest.
type ExportOptions struct {
	// ProfileID selects the ATB obligation profile recorded in verify_report.
	ProfileID string
	// SubmitterRef is an optional caller-controlled reference for the submitting actor.
	SubmitterRef string
	// SubmittedAt overrides the export timestamp. Zero uses time.Now().UTC().
	SubmittedAt time.Time
}

// BundleExport is the ATB-to-custody wire object described in docs/custos-handoff.md.
type BundleExport struct {
	ExportVersion string         `json:"export_version"`
	ReceiptID     string         `json:"receipt_id"`
	BundleHash    string         `json:"bundle_hash"`
	SubmittedAt   string         `json:"submitted_at"`
	ProfileID     string         `json:"profile_id,omitempty"`
	SubmitterRef  string         `json:"submitter_ref,omitempty"`
	VerifyReport  VerifierReport `json:"verify_report"`
	Bundle        []byte         `json:"bundle"`
}

// NewBundleExport loads bundlePath through LoadVerified and returns the custody
// handoff object without contacting any hosted service.
func NewBundleExport(bundlePath string, opts ExportOptions) (BundleExport, error) {
	if bundlePath == "" {
		return BundleExport{}, fmt.Errorf("%w: bundle path is required", ErrInvalidExport)
	}

	verified, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		return BundleExport{}, fmt.Errorf("custody: load verified bundle: %w", err)
	}
	raw, err := os.ReadFile(bundlePath) // #nosec G304 -- caller supplies the local bundle path to export.
	if err != nil {
		return BundleExport{}, fmt.Errorf("custody: read verified bundle bytes: %w", err)
	}

	submittedAt := opts.SubmittedAt
	if submittedAt.IsZero() {
		submittedAt = time.Now().UTC()
	}
	headHash := HeadHash(verified)
	report := verify.Verify(verified, bundlePath, opts.ProfileID)

	return BundleExport{
		ExportVersion: BundleExportVersion,
		ReceiptID:     receiptIDPrefix + headHash,
		BundleHash:    headHash,
		SubmittedAt:   submittedAt.UTC().Format(time.RFC3339),
		ProfileID:     opts.ProfileID,
		SubmitterRef:  opts.SubmitterRef,
		VerifyReport:  verify.ReportFromVerify(report),
		Bundle:        raw,
	}, nil
}

// Export serialises the handoff object to JSON.
func (e BundleExport) Export() ([]byte, error) {
	if e.ExportVersion == "" || e.ReceiptID == "" || e.BundleHash == "" || e.SubmittedAt == "" || len(e.Bundle) == 0 {
		return nil, fmt.Errorf("%w: missing required field", ErrInvalidExport)
	}
	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("custody: serialise bundle export: %w", err)
	}
	return data, nil
}
