// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	verifypkg "github.com/pcguest/atb/internal/verify"
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
			name: "all equals options",
			args: []string{
				"--host=localhost",
				"--port=6060",
				"--bundle=bundle.atb",
				"--profile=atb.profile.rag_answer",
				"--session-token=token",
				"--oidc-issuer=https://issuer.example",
				"--oidc-audience=atb-viewer",
				"--no-open",
				"--log-reveals",
			},
			want: viewConfig{
				BundlePath:   "bundle.atb",
				Host:         "localhost",
				Port:         6060,
				PortSet:      true,
				NoOpen:       true,
				LogReveals:   true,
				ProfilePath:  "atb.profile.rag_answer",
				SessionToken: "token",
				OIDCIssuer:   "https://issuer.example",
				OIDCAudience: "atb-viewer",
			},
		},
		{
			name:    "invalid port",
			args:    []string{"--port", "abc"},
			wantErr: true,
		},
		{name: "help", args: []string{"--help"}, wantErr: true},
		{name: "missing host", args: []string{"--host"}, wantErr: true},
		{name: "empty host", args: []string{"--host="}, wantErr: true},
		{name: "missing port", args: []string{"--port"}, wantErr: true},
		{name: "port too low", args: []string{"--port=0"}, wantErr: true},
		{name: "port too high", args: []string{"--port=65536"}, wantErr: true},
		{name: "missing bundle", args: []string{"--bundle"}, wantErr: true},
		{name: "duplicate bundle flags", args: []string{"--bundle=one", "--bundle=two"}, wantErr: true},
		{name: "duplicate positional bundle", args: []string{"one", "two"}, wantErr: true},
		{name: "missing profile", args: []string{"--profile"}, wantErr: true},
		{name: "missing token", args: []string{"--session-token"}, wantErr: true},
		{name: "missing sessions", args: []string{"--sessions"}, wantErr: true},
		{name: "empty sessions", args: []string{"--sessions="}, wantErr: true},
		{name: "invalid sessions glob", args: []string{"--sessions=["}, wantErr: true},
		{name: "missing OIDC issuer", args: []string{"--oidc-issuer"}, wantErr: true},
		{name: "missing OIDC audience", args: []string{"--oidc-audience"}, wantErr: true},
		{name: "OIDC issuer without audience", args: []string{"--oidc-issuer=https://issuer.example"}, wantErr: true},
		{name: "OIDC audience without issuer", args: []string{"--oidc-audience=atb-viewer"}, wantErr: true},
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
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected config: got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestResolveBundlePathContracts(t *testing.T) {
	if got, err := resolveBundlePath(""); err != nil || got != bundle.DefaultPath() {
		t.Fatalf("empty path = %q, %v", got, err)
	}

	dir := t.TempDir()
	if got, err := resolveBundlePath(dir); err != nil || got != filepath.Join(dir, bundle.BundleFile) {
		t.Fatalf("directory path = %q, %v", got, err)
	}
	file := filepath.Join(dir, "custom.atb")
	if err := os.WriteFile(file, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveBundlePath(file); err != nil || got != file {
		t.Fatalf("file path = %q, %v", got, err)
	}
	missing := filepath.Join(dir, "missing.atb")
	if got, err := resolveBundlePath(missing); err != nil || got != missing {
		t.Fatalf("missing file path = %q, %v", got, err)
	}
	missingDir := filepath.Join(dir, "missing") + string(os.PathSeparator)
	if got, err := resolveBundlePath(missingDir); err != nil || got != filepath.Join(missingDir, bundle.BundleFile) {
		t.Fatalf("missing directory path = %q, %v", got, err)
	}
}

func TestInstallFallbackHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	newInstallFallbackHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "go build -o atb") {
		t.Fatalf("fallback body = %q", rec.Body.String())
	}
}

func TestStartupProfileSummaryMapsVerificationReport(t *testing.T) {
	report := verifypkg.Report{
		Integrity: verifypkg.IntegrityResult{ChainValid: true},
		Anchoring: verifypkg.AnchoringResult{Status: "verified"},
		CAS: &verifypkg.CASResult{
			Overall:            0.88,
			Grade:              "High",
			CorroborationBonus: 0.05,
			EffectiveScore:     0.93,
			SubScores:          map[string]float64{"EC": 0.9},
		},
		Profiles: []verifypkg.ProfileResult{{
			ProfileID:        "atb.profile.policy_decision",
			Version:          1,
			Pass:             false,
			RequiredWarnings: []string{"anchor recommended"},
			CriticalFailures: []verifypkg.CriticalFailure{{
				Kind: "missing_event", Detail: "policy decision missing",
			}},
		}},
		Exclusions:   []string{"network side effects"},
		ResidualRisk: verifypkg.ResidualRisk{Level: "High"},
		ProvabilityGaps: []verifypkg.ProvabilityGap{{
			Gap: "event_coverage", Layer: "L2", Mitigation: "emit events", ClosedWhen: "profile passes",
		}},
	}
	summary := startupProfileSummary(report)
	if summary.ProfileID != "atb.profile.policy_decision" || summary.Pass {
		t.Fatalf("profile summary = %+v", summary)
	}
	if summary.CASGrade != "High" || summary.EffectiveScore != 0.93 || summary.AnchorStatus != "verified" {
		t.Fatalf("CAS summary = %+v", summary)
	}
	if len(summary.CriticalFailures) != 1 || len(summary.ProvabilityGaps) != 1 ||
		len(summary.Exclusions) != 1 || summary.ResidualRiskLevel != "High" {
		t.Fatalf("detail summary = %+v", summary)
	}

	empty := startupProfileSummary(verifypkg.Report{})
	if empty.Pass || empty.CASGrade != "" || empty.CriticalFailures == nil || empty.Warnings == nil {
		t.Fatalf("empty summary = %+v", empty)
	}
}

func TestBuildStartupProfileReports(t *testing.T) {
	b, err := bundle.New()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	summary, report := buildStartupProfileReports(b, path, "atb.profile.policy_decision")
	if summary == nil || report == nil || summary.ProfileID != "atb.profile.policy_decision" {
		t.Fatalf("summary=%+v report=%+v", summary, report)
	}
	summary, report = buildStartupProfileReports(b, path, "does-not-exist")
	if summary != nil || report != nil {
		t.Fatalf("invalid profile returned summary=%+v report=%+v", summary, report)
	}
}

func TestViewSessionPathHelpers(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	first := filepath.Join(root, "first.atb")
	second := filepath.Join(nested, "second.atb")
	ignored := filepath.Join(nested, "ignored.txt")
	for path, content := range map[string]string{first: "one", second: "two", ignored: "ignored"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	paths, err := resolveSessionPaths(root)
	if err != nil {
		t.Fatalf("resolve directory: %v", err)
	}
	if !reflect.DeepEqual(paths, []string{first, second}) {
		t.Fatalf("directory paths = %v", paths)
	}
	globbed, err := resolveSessionPaths(filepath.Join(root, "*.atb"))
	if err != nil || !reflect.DeepEqual(globbed, []string{first}) {
		t.Fatalf("glob paths = %v, %v", globbed, err)
	}
	if _, err := resolveSessionPaths(""); err == nil {
		t.Fatal("empty session path unexpectedly resolved")
	}
	if _, err := resolveSessionPaths("["); err == nil {
		t.Fatal("invalid glob unexpectedly resolved")
	}

	indexed := sessionIndexPaths(first, []string{first, second, second})
	if !reflect.DeepEqual(indexed, []string{first, second}) {
		t.Fatalf("indexed paths = %v", indexed)
	}
	if got := describeSessionSource(nil); got != "" {
		t.Fatalf("empty source = %q", got)
	}
	if got := describeSessionSource([]string{root}); got != root {
		t.Fatalf("directory source = %q", got)
	}
	if got := describeSessionSource([]string{first}); got != filepath.Dir(first) {
		t.Fatalf("file source = %q", got)
	}
	if got := describeSessionSource([]string{first, second}); got != "2 bundle paths" {
		t.Fatalf("multi source = %q", got)
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

	handler, _, tamperDetected, _, err := buildViewServer(bundlePath, false, "", "", "", "")
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

	handler, _, tamperDetected, _, err := buildViewServer(bundlePath, false, "", "", "", "")
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

	handler, _, tamperDetected, openPath, err := buildViewServer(bundlePath, false, "", "", "", "")
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
	if basePort > 65533 {
		t.Skipf("ephemeral port %d leaves no complete two-port fallback range", basePort)
	}

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

func TestPrivacyRevealRecordsToSidecarNotBundle(t *testing.T) {
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
	bundleBefore, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle before reveal: %v", err)
	}

	handler, _, _, _, err := buildViewServer(bundlePath, true, "", "", "", "")
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

	// The authoritative bundle must be byte-for-byte unchanged by the reveal.
	bundleAfter, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle after reveal: %v", err)
	}
	if !bytes.Equal(bundleBefore, bundleAfter) {
		t.Fatalf("authoritative bundle was mutated by reveal")
	}

	// The reveal must be recorded in the sidecar, which verifies independently.
	sidecar, err := bundle.LoadVerified(bundlePath + ".reveals")
	if err != nil {
		t.Fatalf("load reveal sidecar: %v", err)
	}
	if len(sidecar.Records) != 2 {
		t.Fatalf("expected sidecar manifest + 1 reveal, got %d records", len(sidecar.Records))
	}
	auditRecord := sidecar.Records[1]
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

func TestValidateBrowserURLRestrictsLaunchesToLocalHTTP(t *testing.T) {
	for _, rawURL := range []string{
		"http://127.0.0.1:8080/view/",
		"http://localhost:8080/view/",
		"https://[::1]:8080/view/",
	} {
		if err := validateBrowserURL(rawURL); err != nil {
			t.Errorf("validateBrowserURL(%q): %v", rawURL, err)
		}
	}
	for _, rawURL := range []string{
		"https://example.com/",
		"file:///tmp/bundle",
		"https://user:password@localhost/view/",
		"://bad",
	} {
		if err := validateBrowserURL(rawURL); err == nil {
			t.Errorf("validateBrowserURL(%q) unexpectedly succeeded", rawURL)
		}
	}
}

func TestSecurityHeadersCookieUsesTransportSecurity(t *testing.T) {
	handler := withSecurityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "token")
	for _, tc := range []struct {
		name   string
		target string
		secure bool
	}{
		{name: "http", target: "http://localhost/", secure: false},
		{name: "https", target: "https://localhost/", secure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			cookies := rr.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies=%d, want 1", len(cookies))
			}
			cookie := cookies[0]
			if cookie.Secure != tc.secure {
				t.Fatalf("Secure=%v, want %v", cookie.Secure, tc.secure)
			}
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
				t.Fatalf("cookie security attributes: %#v", cookie)
			}
		})
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

	handler, _, _, _, err := buildViewServer(bundlePath, true, "", "", "", "")
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

	handler, _, _, _, err := buildViewServer(bundlePath, false, "", "", "", "")
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

	handler, _, tamperDetected, openPath, err := buildViewServer(bundlePath, false, "", "", "", "")
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
