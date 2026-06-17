// SPDX-License-Identifier: MIT
package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// echoOK is a minimal http.Handler that records invocation and returns 200.
type echoOK struct {
	called bool
}

func (e *echoOK) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	e.called = true
	w.WriteHeader(http.StatusOK)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	t.Parallel()
	next := &echoOK{}
	mw := Middleware("", nil, RoleViewer, next)

	req := httptest.NewRequest(http.MethodGet, "/any-route", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !next.called {
		t.Fatalf("handler not called when token is empty")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_WrongToken(t *testing.T) {
	t.Parallel()
	next := &echoOK{}
	mw := Middleware("correct-token", nil, RoleViewer, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundles", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if next.called {
		t.Fatalf("handler called on wrong token; auth not enforced")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	got := strings.TrimRight(string(body), "\n")
	const want = `{"error":"unauthorised","code":401}`
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestAuthMiddleware_CorrectToken(t *testing.T) {
	t.Parallel()
	next := &echoOK{}
	mw := Middleware("correct-token", nil, RoleViewer, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundles", nil)
	req.Header.Set("Authorization", "Bearer correct-token")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !next.called {
		t.Fatalf("handler not called on correct token")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_HealthBypass(t *testing.T) {
	t.Parallel()
	next := &echoOK{}
	mw := Middleware("correct-token", nil, RoleViewer, next)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !next.called {
		t.Fatalf("handler not called on health bypass")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_CustodyKeyBypass(t *testing.T) {
	t.Parallel()
	// The public verification key must be fetchable without the operator's
	// token, even when auth is enabled — it is not secret and a holder needs it
	// to verify an attestation out-of-band.
	next := &echoOK{}
	mw := Middleware("correct-token", nil, RoleViewer, next)

	req := httptest.NewRequest(http.MethodGet, "/custody/key", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !next.called {
		t.Fatalf("handler not called on custody/key bypass")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_CustodyKeyNonGetStillGuarded(t *testing.T) {
	t.Parallel()
	// The bypass is GET-only; a non-GET to the same path must still require auth.
	next := &echoOK{}
	mw := Middleware("correct-token", nil, RoleViewer, next)

	req := httptest.NewRequest(http.MethodPost, "/custody/key", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if next.called {
		t.Fatalf("POST /custody/key bypassed auth; must be guarded")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
