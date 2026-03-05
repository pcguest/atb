package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantSub string
		wantDay int
		wantErr bool
	}{
		{
			name:    "retention with split flag",
			args:    []string{"retention", "--days", "90"},
			wantSub: "retention",
			wantDay: 90,
		},
		{
			name:    "retention with equals flag",
			args:    []string{"retention", "--days=30"},
			wantSub: "retention",
			wantDay: 30,
		},
		{
			name:    "missing subcommand",
			args:    nil,
			wantErr: true,
		},
		{
			name:    "unknown subcommand",
			args:    []string{"foo", "--days", "10"},
			wantErr: true,
		},
		{
			name:    "missing days value",
			args:    []string{"retention", "--days"},
			wantErr: true,
		},
		{
			name:    "non-numeric days value",
			args:    []string{"retention", "--days", "abc"},
			wantErr: true,
		},
		{
			name:    "zero days",
			args:    []string{"retention", "--days", "0"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"retention", "--wat", "1"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseConfigArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Subcommand != tc.wantSub {
				t.Fatalf("unexpected subcommand: got %q want %q", got.Subcommand, tc.wantSub)
			}
			if got.Days != tc.wantDay {
				t.Fatalf("unexpected days: got %d want %d", got.Days, tc.wantDay)
			}
		})
	}
}

func TestConfigRoundTrip(t *testing.T) {
	now := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
	policy := defaultRetentionPolicy(90, now)
	cfg := atbConfig{
		Version:   configVersion,
		Retention: policy,
	}

	path := filepath.Join(t.TempDir(), ".atb", "config.json")
	if err := saveATBConfig(path, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := loadATBConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.Version != configVersion {
		t.Fatalf("unexpected config version: got %d want %d", loaded.Version, configVersion)
	}
	if loaded.Retention == nil {
		t.Fatalf("expected retention policy")
	}
	if loaded.Retention.Days != 90 {
		t.Fatalf("unexpected retention days: got %d want 90", loaded.Retention.Days)
	}
	if loaded.Retention.ArchiveDir != retentionDefaultArchive {
		t.Fatalf("unexpected archive dir: got %q", loaded.Retention.ArchiveDir)
	}
	if len(loaded.Retention.Scope) != 1 || loaded.Retention.Scope[0] != "run.atb/*.atb" {
		t.Fatalf("unexpected scope: got %#v", loaded.Retention.Scope)
	}
	if loaded.Retention.CutoffBasis != retentionDefaultCutoff {
		t.Fatalf("unexpected cutoff basis: got %q", loaded.Retention.CutoffBasis)
	}
	if loaded.Retention.UpdatedAt != "2026-03-05T10:00:00Z" {
		t.Fatalf("unexpected updated_at: got %q", loaded.Retention.UpdatedAt)
	}
}

func TestConfigLoadRetentionMissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), ".atb", "config.json")
	retention, err := loadRetentionPolicy(missingPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retention != nil {
		t.Fatalf("expected nil retention when config file is missing")
	}

	_, err = loadATBConfig(missingPath)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}
