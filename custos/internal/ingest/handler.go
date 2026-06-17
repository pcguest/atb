// SPDX-License-Identifier: MIT
package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pcguest/atb/pkg/custody"
	// Fixed: The receipt package is internal to the Custos module, not the ATB module.
	"github.com/pcguest/custos/internal/receipt"
)

var (
	// ErrInvalidBundle indicates the uploaded bundle failed hash-chain verification.
	ErrInvalidBundle = errors.New("custos ingest: invalid bundle")
	// ErrEmptyBody indicates the request carried no bundle bytes.
	ErrEmptyBody = errors.New("custos ingest: empty bundle body")
	// Fixed: Nil stores must be reported as configuration errors instead of panicking.
	ErrStoreNotConfigured = errors.New("custos ingest: receipt stores not configured")
)

// IngestHandler validates bundle integrity before custody accept.
type IngestHandler struct {
	ProfileID    string
	WORMStore    receipt.WORMStore
	ReceiptStore receipt.ReceiptStore
	// Signer, when set, attaches a Custos custody attestation to each receipt.
	Signer *receipt.Signer
}

// Handle reads bundle bytes, verifies the hash chain via ATB custody export, and returns the head hash.
func (h IngestHandler) Handle(ctx context.Context, r io.Reader) (*receipt.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Fixed: Store dependencies are required before any verification or persistence path can run.
	if h.WORMStore == nil || h.ReceiptStore == nil {
		return nil, ErrStoreNotConfigured
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("custos ingest: read bundle: %w", err)
	}
	if len(raw) == 0 {
		return nil, ErrEmptyBody
	}

	tmp, err := os.CreateTemp("", "custos-ingest-*.atb")
	if err != nil {
		return nil, fmt.Errorf("custos ingest: temp file: %w", err)
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("custos ingest: write temp bundle: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("custos ingest: close temp bundle: %w", err)
	}

	export, err := custody.NewBundleExport(path, custody.ExportOptions{ProfileID: h.ProfileID})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBundle, err)
	}

	// The WORM store is content-addressed by the SHA-256 of the stored bytes —
	// this is the storage key and the integrity self-check at the storage
	// boundary. It is NOT the bundle's hash-chain head hash (export.BundleHash),
	// which is a different value derived from the last record. Conflating the two
	// made every filesystem ingest fail; address storage by the content hash and
	// keep the chain-head hash as the receipt's BundleHash integrity anchor.
	sum := sha256.Sum256(raw)
	contentHash := hex.EncodeToString(sum[:])
	if _, err := h.WORMStore.Store(ctx, raw, contentHash); err != nil {
		return nil, fmt.Errorf("custos ingest: store bundle in WORM: %w", err)
	}
	// Receipt ID is the content-address of the stored bytes (URL-safe), matching
	// the WORM file name so the bundle is retrievable by receipt ID.
	receiptID := fmt.Sprintf("sha256-%s", contentHash)

	// Marshal VerifierReport to json.RawMessage
	// Fixed: BundleExport exposes VerifyReport as the public custody report field.
	verifierReportJSON, err := json.Marshal(export.VerifyReport)
	if err != nil {
		return nil, fmt.Errorf("custos ingest: marshal verifier report: %w", err)
	}

	// Create Receipt
	newReceipt := receipt.Receipt{
		ExportVersion: "atb.custody.bundle_export.v1", // As per docs/custos-handoff.md
		ReceiptID:     receiptID,
		BundleHash:    export.BundleHash,
		SubmittedAt:   time.Now().UTC(),
		ProfileID:     export.ProfileID,
		VerifyReport:  verifierReportJSON,
	}

	// Attest receipt (independent Custos proof of receipt) before storing.
	if h.Signer != nil {
		att := h.Signer.Attest(newReceipt, time.Now())
		newReceipt.Attestation = &att
	}

	// Store Receipt
	if err := h.ReceiptStore.StoreReceipt(ctx, newReceipt); err != nil {
		return nil, fmt.Errorf("custos ingest: store receipt: %w", err)
	}

	return &newReceipt, nil
}
