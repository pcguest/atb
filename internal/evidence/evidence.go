// SPDX-License-Identifier: MIT
// Package evidence builds structured bundle evidence summaries for auditors
// and CI without changing ATB's local bundle trust boundary.
package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

type SignatureEvidence struct {
	Sequence   int    `json:"sequence"`
	Backend    string `json:"backend"`
	KeyID      string `json:"key_id,omitempty"`
	SignedAt   string `json:"signed_at,omitempty"`
	PubKey     string `json:"pubkey"`
	BundleHash string `json:"bundle_hash"`
	Valid      bool   `json:"valid"`
}

type SnapshotEvidence struct {
	Sequence    int    `json:"sequence"`
	Name        string `json:"name"`
	BundleHash  string `json:"bundle_hash"`
	RecordCount int    `json:"record_count"`
	SnapshotAt  string `json:"snapshot_at"`
}

type ManifestEvidence struct {
	Version   int    `json:"version"`
	BundleID  string `json:"bundle_id"`
	CreatedAt string `json:"created_at"`
}

type BundleEvidence struct {
	Path        string              `json:"path"`
	Manifest    ManifestEvidence    `json:"manifest"`
	Snapshots   []SnapshotEvidence  `json:"snapshots"`
	Signatures  []SignatureEvidence `json:"signatures"`
	RecordCount int                 `json:"record_count"`
	Tampered    bool                `json:"tampered"`
}

// Build loads the bundle at path, verifies it, and extracts structured evidence.
// If verification fails due to tampering, Build still returns evidence with
// Tampered set to true; the error is non-nil in that case.
// Any other load/IO error is returned as-is with a zero BundleEvidence.
func Build(ctx context.Context, path string) (BundleEvidence, error) {
	if err := ctx.Err(); err != nil {
		return BundleEvidence{}, err
	}

	b, err := bundle.Load(path)
	if err != nil {
		return BundleEvidence{}, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = filepath.Clean(path)
	}

	ev := BundleEvidence{
		Path:        absPath,
		Snapshots:   []SnapshotEvidence{},
		Signatures:  []SignatureEvidence{},
		RecordCount: len(b.Records),
	}
	ev.Manifest = manifestEvidence(b.Manifest())

	verifyErr := b.Verify()
	ev.Tampered = verifyErr != nil

	for _, record := range b.Records {
		if err := ctx.Err(); err != nil {
			return BundleEvidence{}, err
		}
		if record.Event.Type != event.TypeSnapshot {
			continue
		}
		if snapshot, ok := snapshotEvidence(record); ok {
			ev.Snapshots = append(ev.Snapshots, snapshot)
		}
	}

	report := verifypkg.Verify(b, absPath, "")
	for _, signature := range report.Signatures {
		ev.Signatures = append(ev.Signatures, SignatureEvidence{
			Sequence:   signature.Sequence,
			Backend:    defaultBackend(signature.Backend),
			KeyID:      signature.KeyID,
			SignedAt:   signature.SignedAt,
			PubKey:     signature.PubKey,
			BundleHash: signature.BundleHash,
			Valid:      signature.Valid,
		})
	}

	if verifyErr != nil {
		return ev, fmt.Errorf("evidence: verify bundle: %w", verifyErr)
	}
	return ev, nil
}

func manifestEvidence(manifest *bundle.ManifestData) ManifestEvidence {
	if manifest == nil {
		return ManifestEvidence{}
	}
	version, _ := strconv.Atoi(manifest.Version)
	return ManifestEvidence{
		Version:   version,
		BundleID:  manifest.BundleID,
		CreatedAt: manifest.CreatedAt,
	}
}

func snapshotEvidence(record bundle.Record) (SnapshotEvidence, bool) {
	var snapshot SnapshotEvidence
	if err := remarshal(record.Event.Data, &snapshot); err != nil {
		return SnapshotEvidence{}, false
	}
	snapshot.Sequence = record.Event.Sequence
	return snapshot, true
}

func defaultBackend(backend string) string {
	if backend == "" {
		return "local"
	}
	return backend
}

func remarshal(input any, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
