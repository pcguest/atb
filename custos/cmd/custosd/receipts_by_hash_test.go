// SPDX-License-Identifier: MIT
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pcguest/custos/internal/receipt"
)

type byHashResponse struct {
	BundleHash string            `json:"bundle_hash"`
	Count      int               `json:"count"`
	Receipts   []receipt.Receipt `json:"receipts"`
}

func getByHash(t *testing.T, mux http.Handler, query string) (int, byHashResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/receipts/by-hash"+query, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var resp byHashResponse
	if rec.Body.Len() > 0 && rec.Header().Get("Content-Type") == "application/json" {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v (body=%q)", err, rec.Body.String())
		}
	}
	return rec.Code, resp
}

func TestFindReceiptsByHashReturnsAllMatches(t *testing.T) {
	mux, store, signer := newSeededMux(t)
	// Two distinct receipts custodying the same bundle hash, one with another.
	seedAttestedReceipt(t, store, signer, "sha256-aaa", "head-shared")
	seedAttestedReceipt(t, store, signer, "sha256-bbb", "head-shared")
	seedAttestedReceipt(t, store, signer, "sha256-ccc", "head-other")

	code, resp := getByHash(t, mux, "?bundle_hash=head-shared")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if resp.BundleHash != "head-shared" {
		t.Fatalf("bundle_hash = %q, want head-shared", resp.BundleHash)
	}
	if resp.Count != 2 || len(resp.Receipts) != 2 {
		t.Fatalf("count = %d / len = %d, want 2", resp.Count, len(resp.Receipts))
	}
	for _, r := range resp.Receipts {
		if r.BundleHash != "head-shared" {
			t.Fatalf("returned receipt with bundle_hash %q", r.BundleHash)
		}
	}
}

func TestFindReceiptsByHashNoMatchIsEmpty(t *testing.T) {
	mux, store, signer := newSeededMux(t)
	seedAttestedReceipt(t, store, signer, "sha256-aaa", "head-1")

	code, resp := getByHash(t, mux, "?bundle_hash=head-unknown")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if resp.Count != 0 || len(resp.Receipts) != 0 {
		t.Fatalf("count = %d, want 0", resp.Count)
	}
}

func TestFindReceiptsByHashRequiresParam(t *testing.T) {
	mux, _, _ := newSeededMux(t)
	code, _ := getByHash(t, mux, "")
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestFindReceiptsByHashRejectsNonGet(t *testing.T) {
	mux, _, _ := newSeededMux(t)
	req := httptest.NewRequest(http.MethodPost, "/receipts/by-hash?bundle_hash=head-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

// TestFindByHashNotShadowedByIDSubtree guards the routing contract: the exact
// /receipts/by-hash pattern must win over the /receipts/ id subtree, so
// "by-hash" is never treated as a receipt ID.
func TestFindByHashNotShadowedByIDSubtree(t *testing.T) {
	mux, store, signer := newSeededMux(t)
	seedAttestedReceipt(t, store, signer, "sha256-aaa", "head-1")

	code, resp := getByHash(t, mux, "?bundle_hash=head-1")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (by-hash route shadowed?)", code)
	}
	if resp.Count != 1 {
		t.Fatalf("count = %d, want 1", resp.Count)
	}
}
