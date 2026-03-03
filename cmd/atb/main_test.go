package main

import (
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestParseAppendPayload(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "single json", args: []string{"{\"ok\":true}"}, want: "{\"ok\":true}"},
		{name: "data flag", args: []string{"--data", "{\"ok\":true}"}, want: "{\"ok\":true}"},
		{name: "missing", args: nil, wantErr: true},
		{name: "invalid", args: []string{"--data"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAppendPayload(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected payload: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeBundlePath(t *testing.T) {
	tmp := t.TempDir()
	if got := normalizeBundlePath(tmp); got != filepath.Join(tmp, bundle.BundleFile) {
		t.Fatalf("expected directory path to resolve to bundle file, got %q", got)
	}

	custom := filepath.Join(tmp, "custom.atb")
	if got := normalizeBundlePath(custom); got != custom {
		t.Fatalf("expected file path to remain unchanged, got %q", got)
	}
}
