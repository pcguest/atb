package mortise

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

	client, err := NewHTTPClient(srv.URL+"/", "secret-token")
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	receipt, err := client.SendBundle(context.Background(), []byte(bundle))
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
	if !strings.Contains(string(receipt.Raw), `"checkpoint":"cp"`) {
		t.Fatalf("raw receipt was not preserved: %s", receipt.Raw)
	}
}

func TestSendBundleReturnsErrorOnRejection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bundle hash chain is not intact or could not be verified", http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	client, err := NewHTTPClient(srv.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SendBundle(context.Background(), []byte("not a bundle"))
	if err == nil {
		t.Fatal("expected an error when Mortise rejects the bundle")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Fatalf("expected the rejection status in the error, got %v", err)
	}
}

func TestNewHTTPClientValidatesEndpoint(t *testing.T) {
	for _, endpoint := range []string{
		"",
		"mortise.example",
		"ftp://mortise.example",
		"https://user:pass@mortise.example",
		"https://mortise.example?token=secret",
		"https://mortise.example/#fragment",
	} {
		if _, err := NewHTTPClient(endpoint, ""); err == nil {
			t.Errorf("NewHTTPClient(%q) unexpectedly succeeded", endpoint)
		}
	}
}

func TestSendBundleValidatesReceiptContract(t *testing.T) {
	for name, response := range map[string]string{
		"version": `{"receipt_version":"unknown","receipt_id":"r","bundle_hash":"h"}`,
		"id":      `{"receipt_version":"custos.receipt.v1","bundle_hash":"h"}`,
		"hash":    `{"receipt_version":"custos.receipt.v1","receipt_id":"r"}`,
		"json":    `{`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, response)
			}))
			defer srv.Close()
			client, err := NewHTTPClient(srv.URL, "")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.SendBundle(context.Background(), []byte("bundle")); err == nil {
				t.Fatal("invalid receipt accepted")
			}
		})
	}
}

func TestSendBundleDoesNotForwardCredentialsAcrossRedirects(t *testing.T) {
	var redirected bool
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected = true
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	client, err := NewHTTPClient(source.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SendBundle(context.Background(), []byte("bundle"))
	if err == nil || !strings.Contains(err.Error(), "307") {
		t.Fatalf("redirect error=%v", err)
	}
	if redirected {
		t.Fatal("Mortise client followed a redirect with credentials")
	}
}
