package main

import (
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
			want: viewConfig{Port: 8080},
		},
		{
			name: "path and port",
			args: []string{"trace.atb", "--port", "9090"},
			want: viewConfig{BundlePath: "trace.atb", Port: 9090},
		},
		{
			name: "port first",
			args: []string{"--port=7070", "run.atb"},
			want: viewConfig{BundlePath: "run.atb", Port: 7070},
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

	handler, _, err := buildViewHandler(bundlePath)
	if err != nil {
		t.Fatalf("buildViewHandler error: %v", err)
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
}
