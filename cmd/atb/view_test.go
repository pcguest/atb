package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestParseViewArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		want    viewConfig
	}{
		{
			name: "default config",
			args: nil,
			want: viewConfig{Host: defaultViewHost, Port: 8080},
		},
		{
			name: "path and port",
			args: []string{"trace.atb", "--port", "9090"},
			want: viewConfig{BundlePath: "trace.atb", Host: defaultViewHost, Port: 9090, PortSet: true},
		},
		{
			name: "bundle flag and no-open",
			args: []string{"--bundle", "run.atb/bundle.atb", "--no-open", "--log-reveals"},
			want: viewConfig{
				BundlePath: "run.atb/bundle.atb",
				Host:       defaultViewHost,
				Port:       8080,
				NoOpen:     true,
				LogReveals: true,
			},
		},
		{
			name: "host override",
			args: []string{"--host", "0.0.0.0"},
			want: viewConfig{
				Host: "0.0.0.0",
				Port: 8080,
			},
		},
		{
			name: "port first",
			args: []string{"--port=7070", "run.atb"},
			want: viewConfig{BundlePath: "run.atb", Host: defaultViewHost, Port: 7070, PortSet: true},
		},
		{
			name:    "invalid port",
			args:    []string{"--port", "abc"},
			wantErr: true,
		},
		{
			// The removed --ui-experimental flag must now be rejected as unknown.
			name:    "unknown flag rejected",
			args:    []string{"--ui-experimental"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseViewArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseViewArgs returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected config: got %+v want %+v", got, tc.want)
			}
		})
	}
}

// TestBuildViewServerFallbackExposesNoData verifies that when the embedded dashboard
// is absent (the typical case in test builds), GET / returns a minimal guidance page
// that does not expose any bundle event data.
func TestBuildViewServerFallbackExposesNoData(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{
		"email":  "secret@example.com",
		"prompt": "do something sensitive",
	})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	handler, _, tamperDetected, _, err := buildViewServer(bundlePath, false, "", "")
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}
	if tamperDetected {
		t.Fatalf("did not expect tamper mode for valid bundle")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// The page must not expose raw event data regardless of the response type.
	body := rr.Body.String()
	if strings.Contains(body, "secret@example.com") {
		t.Fatalf("response must not expose event data (found email in body)")
	}
	if strings.Contains(body, "do something sensitive") {
		t.Fatalf("response must not expose event data (found prompt text in body)")
	}

	// The verification API endpoint must still be served independently.
	verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/verification", nil)
	verifyRR := httptest.NewRecorder()
	handler.ServeHTTP(verifyRR, verifyReq)
	if verifyRR.Code != http.StatusOK {
		t.Fatalf("unexpected verification status: got %d want %d", verifyRR.Code, http.StatusOK)
	}
	if !strings.Contains(verifyRR.Body.String(), `"status":"valid"`) {
		t.Fatalf("expected valid verification response, got %s", verifyRR.Body.String())
	}
}

func TestBuildViewServerTamperMode(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{"prompt": "x"})
	if len(b.Records) == 0 {
		t.Fatalf("expected at least one record")
	}
	b.Records[0].Hash = strings.Repeat("0", 64)
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save tampered bundle: %v", err)
	}

	handler, _, tamperDetected, _, err := buildViewServer(bundlePath, false, "", "")
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}
	if !tamperDetected {
		t.Fatalf("expected tamper mode for invalid bundle")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "TAMPER DETECTED") {
		t.Fatalf("expected tamper warning page")
	}

	verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/verification", nil)
	verifyRR := httptest.NewRecorder()
	handler.ServeHTTP(verifyRR, verifyReq)
	if verifyRR.Code != http.StatusOK {
		t.Fatalf("unexpected verification status: got %d want %d", verifyRR.Code, http.StatusOK)
	}
	if !strings.Contains(verifyRR.Body.String(), `"status":"invalid"`) {
		t.Fatalf("expected invalid verification response, got %s", verifyRR.Body.String())
	}

	metaReq := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/meta", nil)
	metaRR := httptest.NewRecorder()
	handler.ServeHTTP(metaRR, metaReq)
	if metaRR.Code != http.StatusForbidden {
		t.Fatalf("unexpected meta status: got %d want %d", metaRR.Code, http.StatusForbidden)
	}
}

// TestBuildViewServerTamperModeCatchesAllRoutes verifies that a tampered bundle
// serves the tamper warning for any path, not just /.
func TestBuildViewServerTamperModeCatchesAllRoutes(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{"prompt": "x"})
	if len(b.Records) == 0 {
		t.Fatalf("expected at least one record")
	}
	b.Records[0].Hash = strings.Repeat("0", 64)
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save tampered bundle: %v", err)
	}

	handler, _, tamperDetected, openPath, err := buildViewServer(bundlePath, false, "", "")
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}
	if !tamperDetected {
		t.Fatalf("expected tamper mode for invalid bundle")
	}
	if openPath != "/" {
		t.Fatalf("expected tampered bundle to open at /, got %q", openPath)
	}

	for _, path := range []string{"/", "/view/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if !strings.Contains(rr.Body.String(), "TAMPER DETECTED") {
			t.Fatalf("expected tamper warning page at %s, got %s", path, rr.Body.String())
		}
	}
}

func TestCandidateViewPorts(t *testing.T) {
	t.Run("explicit", func(t *testing.T) {
		got := candidateViewPorts(8080, true)
		want := []int{8080}
		if len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("unexpected ports: got=%v want=%v", got, want)
		}
	})

	t.Run("fallback range", func(t *testing.T) {
		got := candidateViewPorts(8080, false)
		want := []int{8080, 8081, 8082}
		if len(got) != len(want) {
			t.Fatalf("unexpected ports length: got=%v want=%v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("unexpected ports: got=%v want=%v", got, want)
			}
		}
	})
}

func TestListenViewPortBindsLoopbackByDefault(t *testing.T) {
	ln, _, err := listenViewPort(defaultViewHost, 0, true)
	if err != nil {
		t.Fatalf("listenViewPort returned error: %v", err)
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener addr type: %T", ln.Addr())
	}
	if addr.IP == nil || !addr.IP.IsLoopback() {
		t.Fatalf("expected loopback bind address, got %v", addr.IP)
	}
	if addr.IP.String() != defaultViewHost {
		t.Fatalf("expected bind address %s, got %s", defaultViewHost, addr.IP.String())
	}
}

func TestListenViewPortFallsBackWhenBusy(t *testing.T) {
	baseListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind local test listener in this environment: %v", err)
	}
	defer baseListener.Close()

	basePort := baseListener.Addr().(*net.TCPAddr).Port

	ln, gotPort, err := listenViewPort(defaultViewHost, basePort, false)
	if err != nil {
		t.Fatalf("listenViewPort returned error: %v", err)
	}
	defer ln.Close()

	if gotPort == basePort {
		t.Fatalf("expected fallback from busy port %d", basePort)
	}
	if gotPort != basePort+1 && gotPort != basePort+2 {
		t.Fatalf("unexpected fallback port: got %d want %d or %d", gotPort, basePort+1, basePort+2)
	}
}

func TestListenViewPortExplicitPortBusy(t *testing.T) {
	baseListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind local test listener in this environment: %v", err)
	}
	defer baseListener.Close()

	basePort := baseListener.Addr().(*net.TCPAddr).Port

	_, _, err = listenViewPort(defaultViewHost, basePort, true)
	if err == nil {
		t.Fatalf("expected error for busy explicit port")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", basePort)) {
		t.Fatalf("expected error to include port %d, got %v", basePort, err)
	}
}

func TestIsAddrInUseError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unix style",
			err:  errors.New("listen tcp 127.0.0.1:8080: bind: address already in use"),
			want: true,
		},
		{
			name: "windows style",
			err:  errors.New("listen tcp 127.0.0.1:8080: bind: Only one usage of each socket address (protocol/network address/port) is normally permitted."),
			want: true,
		},
		{
			name: "different error",
			err:  errors.New("listen tcp 127.0.0.1:8080: bind: permission denied"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAddrInUseError(tc.err); got != tc.want {
				t.Fatalf("isAddrInUseError()=%v want %v for err=%v", got, tc.want, tc.err)
			}
		})
	}
}

func TestPrivacyRevealAppendsAuditEventToBundle(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{
		"email":  "auditor@example.com",
		"prompt": "hello",
	})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	handler, _, _, _, err := buildViewServer(bundlePath, true, "", "")
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}

	// Trigger a response on / to receive the reveal auth cookie from withSecurityHeaders.
	seedReq := httptest.NewRequest(http.MethodGet, "/", nil)
	seedRR := httptest.NewRecorder()
	handler.ServeHTTP(seedRR, seedReq)

	var revealCookie *http.Cookie
	for _, cookie := range seedRR.Result().Cookies() {
		if cookie.Name == "atb_reveal_token" {
			revealCookie = cookie
			break
		}
	}
	if revealCookie == nil || strings.TrimSpace(revealCookie.Value) == "" {
		t.Fatalf("expected reveal auth cookie to be set")
	}

	payload := []byte(`{"seq":1,"field_path":"email","reason":"qa_test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(revealCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected reveal status: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "auditor@example.com") {
		t.Fatalf("expected revealed value in response, got %s", rr.Body.String())
	}

	updated, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load updated bundle: %v", err)
	}
	if len(updated.Records) != 3 {
		t.Fatalf("expected reveal audit append to bundle: got %d records", len(updated.Records))
	}
	auditRecord := updated.Records[2]
	if auditRecord.Event.Type != "privacy.reveal" {
		t.Fatalf("expected privacy.reveal event type, got %q", auditRecord.Event.Type)
	}
	auditData, ok := auditRecord.Event.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected audit event data map, got %T", auditRecord.Event.Data)
	}
	if auditData["field_path"] != "email" {
		t.Fatalf("expected field path in audit event, got %v", auditData["field_path"])
	}
}

func TestPrivacyRevealRequiresAuth(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{
		"email": "auditor@example.com",
	})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	handler, _, _, _, err := buildViewServer(bundlePath, true, "", "")
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}

	payload := []byte(`{"seq":1,"field_path":"email","reason":"qa_test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected reveal status without auth: got %d want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestBuildViewServerSetsSecurityHeaders(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{"prompt": "hello"})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	handler, _, _, _, err := buildViewServer(bundlePath, false, "", "")
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("expected Content-Security-Policy header")
	}
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options nosniff")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY")
	}
}

// TestBuildViewServerServesViewRoute verifies the routing behaviour based on whether
// the embedded dashboard is present.
func TestBuildViewServerServesViewRoute(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "agent.prompt", map[string]interface{}{"prompt": "hello"})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	handler, _, tamperDetected, openPath, err := buildViewServer(bundlePath, false, "", "")
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}
	if tamperDetected {
		t.Fatalf("did not expect tamper mode for valid bundle")
	}

	if _, available := embeddedDashboardFS(); available {
		// With the embedded dashboard the server opens at /view/.
		if openPath != "/view/" {
			t.Fatalf("expected open path /view/ with embedded dashboard, got %q", openPath)
		}
		req := httptest.NewRequest(http.MethodGet, "/view/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("unexpected /view/ status: got %d want %d", rr.Code, http.StatusOK)
		}
	} else {
		// Without the embedded dashboard the server opens at / with install guidance.
		if openPath != "/" {
			t.Fatalf("expected open path / without embedded dashboard, got %q", openPath)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("unexpected / status without embedded dashboard: got %d", rr.Code)
		}
	}
}
