package custos

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendBundlePostsToIngestAndParsesReceipt(t *testing.T) {
	const bundle = "{\"event\":{\"type\":\"atb.bundle.manifest\"}}\n"
	var gotPath, gotAuth string
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"receipt_version": "custos.receipt.v1",
			"receipt_id":      "rcpt-123",
			"bundle_hash":     "abc",
			"content_hash":    "def",
			"leaf_index":      0,
			"checkpoint":      "cp",
		})
	}))
	defer srv.Close()

	receipt, err := NewHTTPClient(srv.URL, "secret-token").SendBundle(context.Background(), []byte(bundle))
	if err != nil {
		t.Fatalf("SendBundle: %v", err)
	}

	if gotPath != "/ingest" {
		t.Fatalf("expected POST to /ingest, got %q", gotPath)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotBody != bundle {
		t.Fatalf("expected the bundle bytes in the request body, got %q", gotBody)
	}
	if receipt.ReceiptID != "rcpt-123" || receipt.BundleHash != "abc" {
		t.Fatalf("receipt not parsed: %+v", receipt)
	}
}

func TestSendBundleReturnsErrorOnRejection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bundle hash chain is not intact or could not be verified", http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	_, err := NewHTTPClient(srv.URL, "").SendBundle(context.Background(), []byte("not a bundle"))
	if err == nil {
		t.Fatal("expected an error when Custos rejects the bundle")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Fatalf("expected the rejection status in the error, got %v", err)
	}
}
