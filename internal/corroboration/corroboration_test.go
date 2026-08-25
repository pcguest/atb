// SPDX-License-Identifier: MIT
package corroboration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/corroboration"
)

func TestHTTPGatewayAdapter_Success(t *testing.T) {
	payload := `{"receipt_id":"rcpt-001","outcome":"success"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	a := &corroboration.HTTPGatewayAdapter{URL: srv.URL, Timeout: 5 * time.Second}
	rec, err := a.Fetch(context.Background(), "abc123hash")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if rec.Source != "http-gateway" {
		t.Errorf("Source = %q, want %q", rec.Source, "http-gateway")
	}
	if rec.ReferenceID != "abc123hash" {
		t.Errorf("ReferenceID = %q, want %q", rec.ReferenceID, "abc123hash")
	}
	if rec.Digest == "" {
		t.Error("Digest must not be empty")
	}
	if rec.RetrievedAt.IsZero() {
		t.Error("RetrievedAt must not be zero")
	}
	if rec.Truncated {
		t.Error("Truncated must be false for small payload")
	}
	if !bytes.Equal(rec.RawEvidence, []byte(payload)) {
		t.Errorf("RawEvidence = %q, want %q", rec.RawEvidence, payload)
	}
}

func TestHTTPGatewayAdapter_FieldMapping(t *testing.T) {
	payload := `{"ok":true}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	a := &corroboration.HTTPGatewayAdapter{URL: srv.URL}
	rec, err := a.Fetch(context.Background(), "ref-xyz")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	data := rec.EventData()
	if data["source"] != "http-gateway" {
		t.Errorf("event data source = %v, want http-gateway", data["source"])
	}
	if data["reference_id"] != "ref-xyz" {
		t.Errorf("event data reference_id = %v, want ref-xyz", data["reference_id"])
	}
	if data["digest"] != rec.Digest {
		t.Errorf("event data digest mismatch")
	}
	if _, ok := data["retrieved_at"].(string); !ok {
		t.Error("event data retrieved_at must be a string")
	}
	raw, ok := data["raw_evidence"].(string)
	if !ok {
		t.Fatal("event data raw_evidence must be a string")
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || string(decoded) != payload {
		t.Errorf("raw_evidence base64 decode = %q, want %q", decoded, payload)
	}
}

func TestHTTPGatewayAdapter_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	a := &corroboration.HTTPGatewayAdapter{URL: srv.URL}
	_, err := a.Fetch(context.Background(), "ref-1")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention status 404, got: %v", err)
	}
}

func TestHTTPGatewayAdapter_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	a := &corroboration.HTTPGatewayAdapter{URL: srv.URL, Timeout: 50 * time.Millisecond}
	_, err := a.Fetch(context.Background(), "ref-1")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestHTTPGatewayAdapter_TruncationAt4KB(t *testing.T) {
	large := strings.Repeat("x", corroboration.MaxRawEvidenceBytes+1)
	payload := `{"data":"` + large + `"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	a := &corroboration.HTTPGatewayAdapter{URL: srv.URL}
	rec, err := a.Fetch(context.Background(), "ref-large")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !rec.Truncated {
		t.Error("Truncated must be true when payload exceeds MaxRawEvidenceBytes")
	}
	if len(rec.RawEvidence) > 0 {
		t.Error("RawEvidence must be nil when payload is truncated")
	}

	data := rec.EventData()
	if data["truncated"] != true {
		t.Error("event data truncated must be true")
	}
	if _, ok := data["raw_evidence"]; ok {
		t.Error("event data must not contain raw_evidence when truncated")
	}
}

func TestHTTPGatewayAdapterRejectsUnsafeURLAndOversizedResponse(t *testing.T) {
	t.Run("credentials", func(t *testing.T) {
		a := &corroboration.HTTPGatewayAdapter{URL: "https://user:secret@example.test"}
		if _, err := a.Fetch(context.Background(), "ref"); err == nil {
			t.Fatal("expected credential-bearing URL rejection")
		}
	})

	t.Run("oversized", func(t *testing.T) {
		payload := `{"data":"` + strings.Repeat("x", corroboration.MaxHTTPResponseBytes) + `"}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(payload))
		}))
		defer srv.Close()

		a := &corroboration.HTTPGatewayAdapter{URL: srv.URL}
		if _, err := a.Fetch(context.Background(), "ref"); err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("expected response-size error, got %v", err)
		}
	})
}
