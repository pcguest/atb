// SPDX-License-Identifier: MIT
package apiv1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

func TestSessionsEndpointsUseConfiguredPrebuiltIndex(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	secondPath, _ := createRichTestBundle(t)
	entries, err := sessionindex.BuildIndex(context.Background(), []string{bundlePath, secondPath})
	if err != nil {
		t.Fatal(err)
	}
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
	if len(sessions.Sessions) != 2 {
		t.Fatalf("sessions count: got %d want %d", len(sessions.Sessions), 2)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/sessions/by-actor", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("by-actor: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	actors := decodeResponseJSON[ActorsResponse](t, rr)
	total := 0
	for _, group := range actors.Actors {
		total += len(group)
	}
	if total != 2 {
		t.Fatalf("actor groups: got %+v", actors.Actors)
	}
	for _, endpoint := range []string{"/api/v1/sessions", "/api/v1/sessions/by-actor"} {
		for _, query := range []string{"", "bundle_dir", "bundle_dir=", "bundle_dir=&bundle_dir=", "Bundle_Dir=ignored", "BUNDLE_DIR=ignored"} {
			t.Run(endpoint+"/"+query, func(t *testing.T) {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, endpoint+"?"+query, nil))
				if rr.Code != http.StatusOK {
					t.Fatalf("configured multi-bundle index lost: status=%d body=%s", rr.Code, rr.Body.String())
				}
				var got []sessionindex.SessionEntry
				if endpoint == "/api/v1/sessions" {
					got = decodeResponseJSON[SessionsResponse](t, rr).Sessions
				} else {
					for _, group := range decodeResponseJSON[ActorsResponse](t, rr).Actors {
						got = append(got, group...)
					}
				}
				paths := map[string]bool{}
				for _, entry := range got {
					paths[entry.BundlePath] = true
				}
				if len(got) != 2 || !paths[bundlePath] || !paths[secondPath] {
					t.Fatalf("configured bundle paths lost: %+v", got)
				}
			})
		}
	}
}

func TestSessionsEndpointsRejectBundleDirOverride(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	bundleDir := filepath.Dir(bundlePath)
	linkPath := filepath.Join(t.TempDir(), "bundle-link")
	if err := os.Symlink(bundleDir, linkPath); err != nil {
		// Some Windows runners cannot create symlinks. The same path must still
		// be rejected without looking up whether it exists or is a symlink.
		t.Logf("symlink creation unavailable; testing the unresolved path: %v", err)
	}
	invalidDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(invalidDir, "invalid.atb"), []byte("synthetic invalid bundle\n"), 0600); err != nil {
		t.Fatal(err)
	}
	const token = "explicit-session-token"
	server, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:   bundlePath,
		Bundle:       b,
		SessionToken: token,
	})
	indexCalls := 0
	server.indexSessions = func(*http.Request) ([]sessionindex.SessionEntry, error) {
		indexCalls++
		return nil, errors.New("indexing must not run for rejected requests")
	}

	paths := []string{"/api/v1/sessions", "/api/v1/sessions/by-actor"}
	invalid := []struct {
		name  string
		query string
	}{
		{"whitespace", "bundle_dir=%20"},
		{"nul", "bundle_dir=%00"},
		{"path_with_spaces", "bundle_dir=path%20cats"},
		{"absolute", "bundle_dir=" + url.QueryEscape(bundleDir)},
		{"traversal", "bundle_dir=../outside"},
		{"glob", "bundle_dir=" + url.QueryEscape(bundleDir+"/*.atb")},
		{"symlink", "bundle_dir=" + url.QueryEscape(linkPath)},
		{"invalid_bundle", "bundle_dir=" + url.QueryEscape(invalidDir)},
		{"missing", "bundle_dir=" + url.QueryEscape(filepath.Join(bundleDir, "missing"))},
		{"malformed_pattern", "bundle_dir=%5B"},
		{"repeated_empty_then_nonempty", "bundle_dir=&bundle_dir=reject"},
		{"repeated_nonempty_then_empty", "bundle_dir=reject&bundle_dir="},
		{"repeated_middle_nonempty", "bundle_dir=&bundle_dir=%09&bundle_dir="},
		{"encoded_parameter", "bundle%5Fdir=reject"},
		{"malformed_query", "bundle_dir=%zz"},
		{"malformed_query_after_empty", "bundle_dir=&bundle_dir=%zz"},
		{"semicolon", "bundle_dir=reject;ignored=x"},
	}
	for _, path := range paths {
		for _, tc := range invalid {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.URL.RawQuery = tc.query
				req.Header.Set(sessionAuthHeader, token)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				if rr.Code != http.StatusBadRequest {
					t.Fatalf("status: got %d want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
				}
				want := `{"error":"invalid query parameters"}`
				if got := strings.TrimSpace(rr.Body.String()); got != want {
					t.Fatalf("body: got %q want %q", got, want)
				}
				if strings.Contains(rr.Body.String(), "reject") || strings.Contains(rr.Body.String(), bundleDir) {
					t.Fatalf("request value leaked in body: %s", rr.Body.String())
				}
				if indexCalls != 0 {
					t.Fatal("rejected query reached filesystem selection/indexing")
				}
				// No token (and a wrong token) must still fail at middleware,
				// without exposing validation details or invoking indexing.
				for _, candidate := range []string{"", "wrong-token"} {
					req.Header.Set(sessionAuthHeader, candidate)
					rr = httptest.NewRecorder()
					handler.ServeHTTP(rr, req)
					if rr.Code != http.StatusUnauthorized || rr.Body.String() != "{\"error\":\"unauthorized\"}\n" || indexCalls != 0 {
						t.Fatalf("authentication changed: status=%d body=%s calls=%d", rr.Code, rr.Body.String(), indexCalls)
					}
				}
			})
		}
	}
	// A positive control makes the seam non-vacuous: missing/empty values
	// reach existing configured indexing instead of the rejection path.
	for _, path := range paths {
		for _, query := range []string{"", "bundle_dir=", "bundle_dir=&bundle_dir="} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.URL.RawQuery = query
			req.Header.Set(sessionAuthHeader, token)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusInternalServerError || rr.Body.String() != "{\"error\":\"session index unavailable\"}\n" {
				t.Fatalf("empty query %q: status=%d body=%s", query, rr.Code, rr.Body.String())
			}
		}
	}
	if indexCalls != 6 {
		t.Fatalf("indexing positive-control calls=%d want=6", indexCalls)
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path+"?session_token="+token+"&bundle_dir=reject", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest || rr.Body.String() != "{\"error\":\"invalid query parameters\"}\n" || indexCalls != 6 {
			t.Fatalf("query-token authentication changed: status=%d calls=%d", rr.Code, indexCalls)
		}
	}
	// Rejection must also precede an already-known verification failure; it
	// must not reveal that state or enter indexing for a rejected override.
	server.verifyErr = errors.New("synthetic verification failure")
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path+"?bundle_dir=reject", nil)
		req.Header.Set(sessionAuthHeader, token)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest || rr.Body.String() != "{\"error\":\"invalid query parameters\"}\n" || indexCalls != 6 {
			t.Fatalf("override reached verification/indexing: status=%d calls=%d", rr.Code, indexCalls)
		}
	}
}

func TestSessionsEndpointsAuthenticationWithoutOverrides(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)
	_, handler := buildTestAPIServer(t, APIConfig{
		BundlePath:   bundlePath,
		Bundle:       b,
		SessionToken: "explicit-session-token",
	})
	for _, path := range []string{"/api/v1/sessions", "/api/v1/sessions/by-actor"} {
		for _, query := range []string{"", "bundle_dir=", "bundle_dir=&bundle_dir="} {
			req := httptest.NewRequest(http.MethodGet, path+"?"+query, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized || rr.Body.String() != "{\"error\":\"unauthorized\"}\n" {
				t.Fatalf("unauthenticated %s: status=%d body=%s", path, rr.Code, rr.Body.String())
			}
		}
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
