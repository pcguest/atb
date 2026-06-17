// SPDX-License-Identifier: MIT
package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pcguest/custos/internal/auth"
	"github.com/pcguest/custos/internal/ingest"
	"github.com/pcguest/custos/internal/receipt"
)

func newTestMux(t *testing.T, maxIngestBytes int64) http.Handler {
	t.Helper()
	worm := receipt.NewInMemoryWORMStore()
	rcpt := receipt.NewInMemoryReceiptStore()
	handler := ingest.IngestHandler{WORMStore: worm, ReceiptStore: rcpt}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return auth.Middleware("", nil, auth.RoleAdmin, newMux(handler, worm, rcpt, maxIngestBytes, logger))
}

func postIngest(t *testing.T, mux http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/ingest", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestIngestRejectsOversizeBody(t *testing.T) {
	mux := newTestMux(t, 16) // 16-byte cap
	rec := postIngest(t, mux, strings.Repeat("x", 4096))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestIngestUnderLimitInvalidBundleIs422(t *testing.T) {
	// A normal-sized but invalid bundle must reach verification (422),
	// proving the size limit is not the blocker for ordinary payloads.
	mux := newTestMux(t, defaultMaxIngestBytes)
	rec := postIngest(t, mux, "this is not a valid atb bundle")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestIngestEmptyBodyIs400(t *testing.T) {
	mux := newTestMux(t, defaultMaxIngestBytes)
	rec := postIngest(t, mux, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestIngestRejectsNonPost(t *testing.T) {
	mux := newTestMux(t, defaultMaxIngestBytes)
	req := httptest.NewRequest(http.MethodGet, "/ingest", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestValidateBindConfig(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		token   string
		wantErr bool
	}{
		{"loopback ipv4 no token", "127.0.0.1", "", false},
		{"localhost no token", "localhost", "", false},
		{"loopback ipv6 no token", "::1", "", false},
		{"all interfaces no token", "0.0.0.0", "", true},
		{"empty host no token", "", "", true},
		{"lan host no token", "192.168.1.5", "", true},
		{"all interfaces with token", "0.0.0.0", "secret", false},
		{"lan host with token", "192.168.1.5", "secret", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBindConfig(tc.host, tc.token)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateBindConfig(%q, %q) err=%v, wantErr=%v", tc.host, tc.token, err, tc.wantErr)
			}
		})
	}
}

func TestNewHTTPServerHasBoundedTimeouts(t *testing.T) {
	srv := newHTTPServer("127.0.0.1:0", http.NewServeMux())
	if srv.ReadHeaderTimeout <= 0 {
		t.Fatal("ReadHeaderTimeout must be set")
	}
	if srv.ReadTimeout <= 0 {
		t.Fatal("ReadTimeout must be set")
	}
	if srv.WriteTimeout <= 0 {
		t.Fatal("WriteTimeout must be set")
	}
	if srv.IdleTimeout <= 0 {
		t.Fatal("IdleTimeout must be set")
	}
	if srv.MaxHeaderBytes <= 0 {
		t.Fatal("MaxHeaderBytes must be set")
	}
}
