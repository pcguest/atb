package main

import (
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestParseTrustReportArgs(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "custom.atb")

	tests := []struct {
		name    string
		args    []string
		want    trustReportConfig
		wantErr bool
	}{
		{
			name: "defaults",
			want: trustReportConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     "markdown",
			},
		},
		{
			name: "json format",
			args: []string{"--format", "json"},
			want: trustReportConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     "json",
			},
		},
		{
			name: "path and equals-format",
			args: []string{custom, "--format=markdown"},
			want: trustReportConfig{
				BundlePath: custom,
				Format:     "markdown",
			},
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
			name:    "too many paths",
			args:    []string{"one.atb", "two.atb"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--wat"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTrustReportArgs(tc.args)
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
				t.Fatalf("unexpected config: got %+v want %+v", got, tc.want)
			}
		})
	}
}
