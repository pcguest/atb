package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
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
			want: viewConfig{Port: 8080},
		},
		{
			name: "path and port",
			args: []string{"trace.atb", "--port", "9090"},
			want: viewConfig{BundlePath: "trace.atb", Port: 9090, PortSet: true},
		},
		{
			name: "bundle flag and no-open",
			args: []string{"--bundle", "run.atb/bundle.atb", "--no-open", "--log-reveals"},
			want: viewConfig{
				BundlePath: "run.atb/bundle.atb",
				Port:       8080,
				NoOpen:     true,
				LogReveals: true,
			},
		},
		{
			name: "port first",
			args: []string{"--port=7070", "run.atb"},
			want: viewConfig{BundlePath: "run.atb", Port: 7070, PortSet: true},
		},
		{
			name:    "invalid port",
			args:    []string{"--port", "abc"},
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

func TestBuildViewHandlerServesTimeline(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := bundle.New()
	if err := b.Append("agent.prompt", map[string]interface{}{
		"timestamp": "2026-03-03T04:00:00Z",
		"actor":     "assistant",
		"prompt":    "Outline launch tasks",
	}); err != nil {
		t.Fatalf("append agent.prompt: %v", err)
	}
	if err := b.Append("snapshot.build", map[string]interface{}{
		"gate": "pass",
	}); err != nil {
		t.Fatalf("append snapshot.build: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	handler, _, tamperDetected, _, err := buildViewServer(bundlePath, false)
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}
	if tamperDetected {
		t.Fatalf("did not expect tamper mode for valid bundle")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	html := rr.Body.String()
	checks := []string{
		"ATB Trace Viewer",
		"snapshot.build",
		"Gate: PASS",
		"Hash chain verified",
		"View JSON + hash details",
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Fatalf("expected HTML to contain %q", want)
		}
	}

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

	b := bundle.New()
	if err := b.Append("agent.prompt", map[string]interface{}{"prompt": "x"}); err != nil {
		t.Fatalf("append agent.prompt: %v", err)
	}
	if len(b.Records) == 0 {
		t.Fatalf("expected at least one record")
	}
	b.Records[0].Hash = strings.Repeat("0", 64)
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save tampered bundle: %v", err)
	}

	handler, _, tamperDetected, _, err := buildViewServer(bundlePath, false)
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

func TestListenViewPortFallsBackWhenBusy(t *testing.T) {
	baseListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind local test listener in this environment: %v", err)
	}
	defer baseListener.Close()

	basePort := baseListener.Addr().(*net.TCPAddr).Port

	ln, gotPort, err := listenViewPort(basePort, false)
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

	_, _, err = listenViewPort(basePort, true)
	if err == nil {
		t.Fatalf("expected error for busy explicit port")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", basePort)) {
		t.Fatalf("expected error to include port %d, got %v", basePort, err)
	}
}

func TestPrivacyRevealWritesAuditLog(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")

	b := bundle.New()
	if err := b.Append("agent.prompt", map[string]interface{}{
		"email":  "auditor@example.com",
		"prompt": "hello",
	}); err != nil {
		t.Fatalf("append agent.prompt: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	handler, _, _, _, err := buildViewServer(bundlePath, true)
	if err != nil {
		t.Fatalf("buildViewServer error: %v", err)
	}

	payload := []byte(`{"seq":1,"field_path":"email","reason":"qa_test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/privacy/reveal", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected reveal status: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "auditor@example.com") {
		t.Fatalf("expected revealed value in response, got %s", rr.Body.String())
	}

	logPath := filepath.Join(tmp, "viewer-audit.log")
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read reveal log: %v", err)
	}
	logLine := string(raw)
	if !strings.Contains(logLine, `"event_type":"ui.privacy.reveal"`) {
		t.Fatalf("expected privacy reveal event type in log, got %s", logLine)
	}
	if !strings.Contains(logLine, `"event_seq":1`) {
		t.Fatalf("expected event sequence in log, got %s", logLine)
	}
	if !strings.Contains(logLine, `"field_path":"email"`) {
		t.Fatalf("expected field path in log, got %s", logLine)
	}
}
