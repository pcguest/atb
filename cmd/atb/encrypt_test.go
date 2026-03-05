package main

import (
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func normalizePathForTest(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func TestParseEncryptArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		wantOut  string
		wantErr  bool
		wantPass string
	}{
		{
			name:     "defaults path",
			args:     []string{"--password", "test123"},
			wantPath: bundle.DefaultPath(),
			wantOut:  bundle.DefaultPath() + ".enc",
			wantPass: "test123",
		},
		{
			name:     "custom path",
			args:     []string{"my.atb", "--password=abc"},
			wantPath: "my.atb",
			wantOut:  "my.atb.enc",
			wantPass: "abc",
		},
		{
			name:    "missing password",
			args:    []string{"my.atb"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--wat", "x"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseEncryptArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if normalizePathForTest(got.InputPath) != normalizePathForTest(tc.wantPath) {
				t.Fatalf("unexpected input path: got %q want %q", got.InputPath, tc.wantPath)
			}
			if normalizePathForTest(got.OutputPath) != normalizePathForTest(tc.wantOut) {
				t.Fatalf("unexpected output path: got %q want %q", got.OutputPath, tc.wantOut)
			}
			if got.Password != tc.wantPass {
				t.Fatalf("unexpected password parse")
			}
		})
	}
}

func TestParseDecryptArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		wantOut  string
		wantPass string
		wantErr  bool
	}{
		{
			name:     "encrypted path with enc suffix",
			args:     []string{"run.atb/bundle.atb.enc", "--password", "test123"},
			wantPath: "run.atb/bundle.atb.enc",
			wantOut:  "run.atb/bundle.atb",
			wantPass: "test123",
		},
		{
			name:     "without enc suffix",
			args:     []string{"custom.bin", "--password=abc"},
			wantPath: "custom.bin",
			wantOut:  "custom.bin.decrypted.atb",
			wantPass: "abc",
		},
		{
			name:    "missing path",
			args:    []string{"--password", "x"},
			wantErr: true,
		},
		{
			name:    "missing password",
			args:    []string{"x.enc"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDecryptArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if normalizePathForTest(got.InputPath) != normalizePathForTest(tc.wantPath) {
				t.Fatalf("unexpected input path: got %q want %q", got.InputPath, tc.wantPath)
			}
			if normalizePathForTest(got.OutputPath) != normalizePathForTest(tc.wantOut) {
				t.Fatalf("unexpected output path: got %q want %q", got.OutputPath, tc.wantOut)
			}
			if got.Password != tc.wantPass {
				t.Fatalf("unexpected password parse")
			}
		})
	}
}
