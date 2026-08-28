// SPDX-License-Identifier: MIT
package apiv1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/sessionindex"
)

func testSessionEntries() []sessionindex.SessionEntry {
	return []sessionindex.SessionEntry{
		{
			SessionID:     "sess-1",
			Actor:         sessionindex.ActorSummary{DisplayName: "alice", Email: "alice@example.com"},
			StartedAt:     time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
			ExchangeCount: 2,
			BundlePath:    "/tmp/sess-1.atb",
		},
		{
			SessionID:     "sess-2",
			Actor:         sessionindex.ActorSummary{DisplayName: "bob"},
			StartedAt:     time.Date(2026, 3, 12, 1, 0, 0, 0, time.UTC),
			ExchangeCount: 1,
			BundlePath:    "/tmp/sess-2.atb",
		},
	}
}

func TestSessionsEndpointsPreIndexed(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	entries := testSessionEntries()
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
		Sessions:        entries,
		SessionsByActor: sessionindex.GroupByActor(entries),
		SessionIndexed:  true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sessions: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	sessions := decodeResponseJSON[SessionsResponse](t, rr)
	if len(sessions.Sessions) != len(entries) {
		t.Fatalf("sessions count: got %d want %d", len(sessions.Sessions), len(entries))
	}
	if sessions.Sessions[0].SessionID != "sess-1" {
		t.Fatalf("session id: got %q want %q", sessions.Sessions[0].SessionID, "sess-1")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions/by-actor", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sessions by actor: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	actors := decodeResponseJSON[ActorsResponse](t, rr)
	if len(actors.Actors) != 2 {
		t.Fatalf("actor groups: got %d want %d", len(actors.Actors), 2)
	}
	if got := len(actors.Actors["alice"]); got != 1 {
		t.Fatalf("alice sessions: got %d want %d", got, 1)
	}

	for _, path := range []string{"/api/v1/sessions", "/api/v1/sessions/by-actor"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("method guard %s: got %d want %d", path, rr.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestSessionsEndpointsBuildIndexFromSelectedBundle(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	bundleDir := filepath.Dir(bundlePath)

	// No pre-built index: handler indexes only the explicitly selected bundle.
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sessions from dir: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	sessions := decodeResponseJSON[SessionsResponse](t, rr)
	if len(sessions.Sessions) != 1 {
		t.Fatalf("sessions from dir count: got %d want %d", len(sessions.Sessions), 1)
	}

	// Explicit bundle_dir override exercises the same scan path via query param.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions/by-actor?bundle_dir="+bundleDir, nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("by-actor with bundle_dir: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	actors := decodeResponseJSON[ActorsResponse](t, rr)
	total := 0
	for _, group := range actors.Actors {
		total += len(group)
	}
	if total != 1 {
		t.Fatalf("by-actor session total: got %d want %d", total, 1)
	}
}

func TestSessionsEndpointsIgnoreSiblingBundlesByDefault(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	tamperedPath := filepath.Join(filepath.Dir(bundlePath), "tampered-sibling.atb")
	if err := os.WriteFile(tamperedPath, []byte("not a valid bundle\n"), 0o600); err != nil {
		t.Fatalf("write tampered sibling: %v", err)
	}

	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("selected bundle sessions: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	sessions := decodeResponseJSON[SessionsResponse](t, rr)
	if len(sessions.Sessions) != 1 {
		t.Fatalf("selected bundle session count: got %d want %d", len(sessions.Sessions), 1)
	}
}

func TestSessionsIndexErrorResponses(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)

	// Load errors from indexing surface as 403 with per-bundle detail.
	loadErr := sessionindex.LoadErrors{{Path: "/tmp/bad.atb", Reason: "hash chain broken"}}
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
		Sessions:        testSessionEntries(),
		SessionsByActor: sessionindex.GroupByActor(testSessionEntries()),
		SessionIndexErr: loadErr,
		SessionIndexed:  true,
	})
	for _, path := range []string{"/api/v1/sessions", "/api/v1/sessions/by-actor"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("load error %s: got %d want %d body=%s", path, rr.Code, http.StatusForbidden, rr.Body.String())
		}
		resp := decodeResponseJSON[SessionIndexErrorResponse](t, rr)
		if len(resp.Bundles) != 1 || resp.Bundles[0].Path != "/tmp/bad.atb" {
			t.Fatalf("load error detail %s: got %+v", path, resp.Bundles)
		}
	}

	// Non-load errors surface as 500 without bundle detail.
	_, genericHandler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
		SessionIndexErr: errors.New("index unavailable"),
		SessionIndexed:  true,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	rr := httptest.NewRecorder()
	genericHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("generic index error: got %d want %d body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
}

func TestBundleVerifyComputesSummary(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundle/verify", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("verify: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	summary := decodeResponseJSON[ProfileReportSummary](t, rr)
	if !summary.ChainValid {
		t.Fatalf("verify summary: chain_valid should be true, body=%s", rr.Body.String())
	}
	if summary.CriticalFailures == nil {
		t.Fatalf("critical_failures must be a non-nil array")
	}
	if summary.Warnings == nil {
		t.Fatalf("warnings must be a non-nil array")
	}

	// The computed summary is stored and served by GET /api/v1/bundle/profile.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/bundle/profile", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("profile after verify: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// And the full verifier report becomes available too.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/bundle/verify/report", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("verify report after verify: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestBundleVerifyRejectsBadProfilePath(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:      bundlePath,
		Bundle:          b,
		RevealAuthToken: "test-token",
		ProfilePath:     filepath.Join(t.TempDir(), "missing-profile.json"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundle/verify", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("verify with bad profile: got %d want %d body=%s", rr.Code, http.StatusUnprocessableEntity, rr.Body.String())
	}
}
