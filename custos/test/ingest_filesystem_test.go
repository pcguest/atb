// SPDX-License-Identifier: MIT
package test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/pkg/custody"
	"github.com/pcguest/custos/internal/ingest"
	"github.com/pcguest/custos/internal/receipt"
)

// TestIngestRoundTripsThroughFilesystemStores exercises the real custody path:
// the FileSystemWORMStore and FileSystemReceiptStore that custosd uses in
// production. Every other custos test uses the in-memory stores, which masked a
// bug where the ingest handler passed the bundle's chain-head hash to the
// content-addressed WORM store — the in-memory store ignored it, but the
// filesystem store rejected every real bundle. This test pins the production
// path so that regression cannot return silently.
func TestIngestRoundTripsThroughFilesystemStores(t *testing.T) {
	dir := t.TempDir()
	wormStore := receipt.NewFileSystemWORMStore(filepath.Join(dir, "worm"))
	receiptStore := receipt.NewFileSystemReceiptStore(filepath.Join(dir, "receipts"), receipt.RetentionPolicy{})

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := receipt.NewSigner(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	handler := ingest.IngestHandler{
		WORMStore:    wormStore,
		ReceiptStore: receiptStore,
		Signer:       signer,
		ProfileID:    "atb.profile.privileged_tool_action",
	}

	bundlePath := filepath.Join("..", "..", "examples", "bundles", "profiles", "privileged_tool_action-pass.atb")
	bundleBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read example bundle: %v", err)
	}

	ctx := context.Background()
	f, err := os.Open(bundlePath)
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	defer f.Close()

	rec, err := handler.Handle(ctx, f)
	if err != nil {
		t.Fatalf("ingest through filesystem stores failed: %v", err)
	}

	// The WORM store is content-addressed: the receipt ID is sha256- of the
	// bundle bytes, and that is what the file is named.
	sum := sha256.Sum256(bundleBytes)
	contentHash := hex.EncodeToString(sum[:])
	wantReceiptID := "sha256-" + contentHash
	if rec.ReceiptID != wantReceiptID {
		t.Errorf("receipt ID = %q, want %q (content-addressed)", rec.ReceiptID, wantReceiptID)
	}

	// The receipt's BundleHash remains the ATB hash-chain head hash — a
	// different value from the content hash, and the integrity anchor.
	export, err := custody.NewBundleExport(bundlePath, custody.ExportOptions{ProfileID: "atb.profile.privileged_tool_action"})
	if err != nil {
		t.Fatalf("expected export: %v", err)
	}
	if rec.BundleHash != export.BundleHash {
		t.Errorf("receipt BundleHash = %q, want chain-head %q", rec.BundleHash, export.BundleHash)
	}
	if rec.BundleHash == contentHash {
		t.Error("chain-head hash and content hash must be distinct values (regression guard)")
	}

	// The stored bytes must round-trip out of the WORM store by receipt ID.
	got, err := wormStore.Retrieve(ctx, rec.ReceiptID)
	if err != nil {
		t.Fatalf("retrieve stored bundle: %v", err)
	}
	if len(got) != len(bundleBytes) {
		t.Errorf("retrieved %d bytes, stored %d", len(got), len(bundleBytes))
	}

	// The attestation must verify against its embedded key.
	if err := receipt.VerifyAttestation(*rec); err != nil {
		t.Errorf("stored receipt attestation does not verify: %v", err)
	}

	// Re-ingesting identical content is idempotent (content-addressed store).
	f2, err := os.Open(bundlePath)
	if err != nil {
		t.Fatalf("reopen bundle: %v", err)
	}
	defer f2.Close()
	rec2, err := handler.Handle(ctx, f2)
	if err != nil {
		t.Fatalf("re-ingest should be idempotent, got: %v", err)
	}
	if rec2.ReceiptID != rec.ReceiptID {
		t.Errorf("idempotent re-ingest changed receipt ID: %q vs %q", rec2.ReceiptID, rec.ReceiptID)
	}
}
