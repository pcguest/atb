package main

import (
	"errors"
	"os"
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

func TestParseDryRunFlag(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantArgs   []string
		wantDryRun bool
	}{
		{
			name:       "no flag",
			args:       []string{"snapshot", "--gate", "pass"},
			wantArgs:   []string{"snapshot", "--gate", "pass"},
			wantDryRun: false,
		},
		{
			name:       "flag at end",
			args:       []string{"snapshot", "--gate", "pass", "--dry-run"},
			wantArgs:   []string{"snapshot", "--gate", "pass"},
			wantDryRun: true,
		},
		{
			name:       "flag at start",
			args:       []string{"--dry-run", "feature", "{\"ok\":true}"},
			wantArgs:   []string{"feature", "{\"ok\":true}"},
			wantDryRun: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotArgs, gotDryRun, err := parseDryRunFlag(tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotDryRun != tc.wantDryRun {
				t.Fatalf("unexpected dry-run: got %v want %v", gotDryRun, tc.wantDryRun)
			}
			if len(gotArgs) != len(tc.wantArgs) {
				t.Fatalf("unexpected args length: got %d want %d", len(gotArgs), len(tc.wantArgs))
			}
			for i := range gotArgs {
				if gotArgs[i] != tc.wantArgs[i] {
					t.Fatalf("unexpected arg at index %d: got %q want %q", i, gotArgs[i], tc.wantArgs[i])
				}
			}
		})
	}
}

func TestAppendToDefaultBundleDryRunDoesNotPersist(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	record, err := appendToDefaultBundle("dev.session", map[string]any{"ok": true}, true)
	if err != nil {
		t.Fatalf("appendToDefaultBundle dry-run error: %v", err)
	}
	if record.Event.Sequence != 1 {
		t.Fatalf("unexpected sequence: got %d want 1", record.Event.Sequence)
	}
	if _, err := os.Stat(bundle.DefaultPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no bundle file written in dry-run mode, stat err=%v", err)
	}
}

func TestAppendToDefaultBundleRejectsCorruptExistingBundle(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.MkdirAll(bundle.BundleDir, 0755); err != nil {
		t.Fatalf("mkdir bundle dir: %v", err)
	}
	if err := os.WriteFile(bundle.DefaultPath(), []byte("{not-json}\n"), 0644); err != nil {
		t.Fatalf("write corrupt bundle: %v", err)
	}

	_, err = appendToDefaultBundle("dev.session", map[string]any{"ok": true}, true)
	if err == nil {
		t.Fatalf("expected error for corrupt existing bundle")
	}

	var loadErr mutationLoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("expected mutationLoadError, got %T", err)
	}
}
