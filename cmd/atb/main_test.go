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

func TestParseVerifyArgs(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "custom.atb")

	tests := []struct {
		name       string
		args       []string
		wantPath   string
		wantFormat string
		wantErr    bool
	}{
		{
			name:       "defaults",
			args:       nil,
			wantPath:   bundle.DefaultPath(),
			wantFormat: verifyFormatText,
		},
		{
			name:       "path only",
			args:       []string{custom},
			wantPath:   custom,
			wantFormat: verifyFormatText,
		},
		{
			name:       "json format flag",
			args:       []string{"--format", "json"},
			wantPath:   bundle.DefaultPath(),
			wantFormat: verifyFormatJSON,
		},
		{
			name:       "json format equals syntax",
			args:       []string{"--format=json"},
			wantPath:   bundle.DefaultPath(),
			wantFormat: verifyFormatJSON,
		},
		{
			name:       "path and format",
			args:       []string{custom, "--format", "json"},
			wantPath:   custom,
			wantFormat: verifyFormatJSON,
		},
		{
			name:    "missing format value",
			args:    []string{"--format"},
			wantErr: true,
		},
		{
			name:    "unknown format",
			args:    []string{"--format", "yaml"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--wat"},
			wantErr: true,
		},
		{
			name:    "too many paths",
			args:    []string{"one.atb", "two.atb"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotFormat, err := parseVerifyArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("unexpected path: got %q want %q", gotPath, tc.wantPath)
			}
			if gotFormat != tc.wantFormat {
				t.Fatalf("unexpected format: got %q want %q", gotFormat, tc.wantFormat)
			}
		})
	}
}
