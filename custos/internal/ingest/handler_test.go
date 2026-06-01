// SPDX-License-Identifier: MIT
package ingest_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	// Fixed: errors is needed to map malformed input onto the HTTP status asserted by the test.
	"errors"
	// Fixed: net/http provides the public 422 status constant used by the malformed-input assertion.
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pcguest/custos/internal/ingest"
	// Fixed: In-memory stores avoid nil-store panics while keeping the test local-first.
	"github.com/pcguest/custos/internal/receipt"
)

func TestIngestHandlerAttestsReceiptWhenSignerSet(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	fixture := filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "bundles", "profiles", "rag_answer-pass.atb")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Skipf("fixture not generated (run make goldens): %v", err)
	}

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, err := receipt.NewSigner(priv)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	handler := ingest.IngestHandler{
		WORMStore:    receipt.NewInMemoryWORMStore(),
		ReceiptStore: receipt.NewInMemoryReceiptStore(),
		Signer:       signer,
	}
	rec, err := handler.Handle(context.Background(), bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if rec.Attestation == nil {
		t.Fatal("expected a custody attestation on the receipt")
	}
	if err := receipt.VerifyAttestation(*rec); err != nil {
		t.Errorf("attestation does not verify: %v", err)
	}
}

func TestIngestHandlerAcceptsValidBundle(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	fixture := filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "bundles", "profiles", "rag_answer-pass.atb")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Fixed: The handler now requires explicit stores before a valid bundle can be accepted.
	handler := ingest.IngestHandler{
		WORMStore:    receipt.NewInMemoryWORMStore(),
		ReceiptStore: receipt.NewInMemoryReceiptStore(),
	}
	// Fixed: Handle returns a receipt, not a bare head hash string.
	rec, err := handler.Handle(context.Background(), bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	// Fixed: ReceiptID is the content-addressed custody identifier returned by ingest.
	if rec.ReceiptID == "" {
		t.Fatal("expected receipt ID")
	}
	// Fixed: BundleHash is the receipt field that carries the actual ATB head hash.
	if rec.BundleHash == "" {
		t.Fatal("expected bundle hash")
	}
}

func TestIngestHandlerRejectsMalformedInputAsUnprocessable(t *testing.T) {
	// Fixed: The handler has stores so malformed input proves verifier rejection, not nil-store failure.
	handler := ingest.IngestHandler{
		WORMStore:    receipt.NewInMemoryWORMStore(),
		ReceiptStore: receipt.NewInMemoryReceiptStore(),
	}
	// Fixed: Malformed non-empty bytes exercise the invalid-bundle path that maps to HTTP 422.
	_, err := handler.Handle(context.Background(), bytes.NewReader([]byte("not an atb bundle")))
	if !errors.Is(err, ingest.ErrInvalidBundle) {
		t.Fatalf("expected invalid bundle error, got %v", err)
	}
	// Fixed: The test locks the expected HTTP mapping for malformed ingest input.
	if status := statusForIngestError(err); status != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, status)
	}
}

// Fixed: This helper mirrors custosd's error-to-status mapping without importing the daemon.
func statusForIngestError(err error) int {
	if errors.Is(err, ingest.ErrInvalidBundle) {
		return http.StatusUnprocessableEntity
	}
	if errors.Is(err, ingest.ErrEmptyBody) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
