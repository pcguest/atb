// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pcguest/custos/internal/ingest"
	"github.com/pcguest/custos/internal/receipt"
)

// newSeededMux builds a mux over in-memory stores and returns the mux plus the
// receipt store and a signer, so a test can seed attested receipts directly
// without ingesting a full bundle.
func newSeededMux(t *testing.T) (*http.ServeMux, receipt.ReceiptStore, *receipt.Signer) {
	t.Helper()
	worm := receipt.NewInMemoryWORMStore()
	rcpt := receipt.NewInMemoryReceiptStore()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := receipt.NewSigner(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	handler := ingest.IngestHandler{WORMStore: worm, ReceiptStore: rcpt, Signer: signer}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newMux(handler, worm, rcpt, defaultMaxIngestBytes, logger), rcpt, signer
}

func seedAttestedReceipt(t *testing.T, store receipt.ReceiptStore, signer *receipt.Signer, id, bundleHash string) receipt.Receipt {
	t.Helper()
	rec := receipt.Receipt{
		ExportVersion: "verify.report.v1",
		ReceiptID:     id,
		BundleHash:    bundleHash,
		SubmittedAt:   time.Now().UTC().Format(time.RFC3339),
		VerifyReport:  json.RawMessage(`{}`),
	}
	att := signer.Attest(rec, time.Now())
	rec.Attestation = &att
	if err := store.Save(context.Background(), rec); err != nil {
		t.Fatalf("seed receipt: %v", err)
	}
	return rec
}

func TestListReceiptsEmpty(t *testing.T) {
	mux, _, _ := newSeededMux(t)
	req := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Count    int               `json:"count"`
		Receipts []receipt.Receipt `json:"receipts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 0 || len(resp.Receipts) != 0 {
		t.Fatalf("want empty custody log, got count=%d len=%d", resp.Count, len(resp.Receipts))
	}
	// receipts must be an empty array, never null, so clients can iterate safely.
	if resp.Receipts == nil {
		t.Error("receipts should serialise as [], not null")
	}
}

func TestListReceiptsEnumeratesCustodyLog(t *testing.T) {
	mux, store, signer := newSeededMux(t)
	seedAttestedReceipt(t, store, signer, "rcpt-1", "hashaaa")
	seedAttestedReceipt(t, store, signer, "rcpt-2", "hashbbb")

	req := httptest.NewRequest(http.MethodGet, "/receipts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Count    int               `json:"count"`
		Receipts []receipt.Receipt `json:"receipts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 2 || len(resp.Receipts) != 2 {
		t.Fatalf("want 2 receipts, got count=%d len=%d", resp.Count, len(resp.Receipts))
	}
	for _, r := range resp.Receipts {
		if r.Attestation == nil {
			t.Errorf("listed receipt %q missing its attestation", r.ReceiptID)
		}
	}
}

func TestListReceiptsRejectsNonGet(t *testing.T) {
	mux, _, _ := newSeededMux(t)
	req := httptest.NewRequest(http.MethodPost, "/receipts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestVerifyAttestationValid(t *testing.T) {
	mux, store, signer := newSeededMux(t)
	seedAttestedReceipt(t, store, signer, "rcpt-ok", "hash-ok")

	req := httptest.NewRequest(http.MethodGet, "/receipts/rcpt-ok/attestation", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var res attestationResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !res.Valid {
		t.Fatalf("attestation should verify, got error=%q", res.Error)
	}
	if res.PubKey == "" || res.Algorithm != "ed25519" {
		t.Errorf("attestation result missing key material: %+v", res)
	}
	if res.BundleHash != "hash-ok" {
		t.Errorf("bundle_hash = %q, want hash-ok", res.BundleHash)
	}
}

func TestVerifyAttestationTamperedIsInvalid(t *testing.T) {
	mux, store, signer := newSeededMux(t)
	rec := seedAttestedReceipt(t, store, signer, "rcpt-bad", "original-hash")
	// Tamper: re-save the receipt with a different bundle hash than the one the
	// attestation was signed over. The signature must no longer verify.
	rec.BundleHash = "tampered-hash"
	if err := store.Save(context.Background(), rec); err != nil {
		t.Fatalf("re-save tampered receipt: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/receipts/rcpt-bad/attestation", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (an invalid attestation is a finding, not a server error)", w.Code)
	}
	var res attestationResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Valid {
		t.Fatal("tampered receipt must not verify")
	}
	if res.Error == "" {
		t.Error("invalid attestation should carry an error explanation")
	}
}

func TestVerifyAttestationMissingReceiptIs404(t *testing.T) {
	mux, _, _ := newSeededMux(t)
	req := httptest.NewRequest(http.MethodGet, "/receipts/does-not-exist/attestation", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
