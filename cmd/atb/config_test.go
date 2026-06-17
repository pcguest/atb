// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/retentionaudit"
)

func TestConfigParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantSub   string
		wantDay   int
		wantAllow bool
		wantErr   bool
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
			name:      "retention with statutory minimum override",
			args:      []string{"retention", "--days", "90", "--allow-below-eu-minimum"},
			wantSub:   "retention",
			wantDay:   90,
			wantAllow: true,
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
			if got.AllowBelowStatutoryMinimum != tc.wantAllow {
				t.Fatalf("unexpected statutory minimum override: got %t want %t", got.AllowBelowStatutoryMinimum, tc.wantAllow)
			}
		})
	}
}

func TestConfigRetentionMinimum(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantCode       int
		wantDays       int
		wantStderrPart string
		wantNoWarning  bool
	}{
		{
			name:          "minimum or greater succeeds without warning",
			args:          []string{"retention", "--days", "183"},
			wantCode:      exitSuccess,
			wantDays:      183,
			wantNoWarning: true,
		},
		{
			name:           "below minimum without override fails",
			args:           []string{"retention", "--days", "90"},
			wantCode:       exitUserError,
			wantStderrPart: "183",
		},
		{
			name:           "below minimum with override succeeds with warning",
			args:           []string{"retention", "--days", "90", "--allow-below-eu-minimum"},
			wantCode:       exitSuccess,
			wantDays:       90,
			wantStderrPart: "WARNING",
		},
		{
			name:           "zero days fails",
			args:           []string{"retention", "--days", "0"},
			wantCode:       exitUserError,
			wantStderrPart: "--days must be > 0",
		},
		{
			name:           "negative days fails",
			args:           []string{"retention", "--days", "-1"},
			wantCode:       exitUserError,
			wantStderrPart: "--days must be > 0",
		},
		{
			name:          "override at minimum succeeds without warning",
			args:          []string{"retention", "--days", "183", "--allow-below-eu-minimum"},
			wantCode:      exitSuccess,
			wantDays:      183,
			wantNoWarning: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			gotCode := runConfig(tc.args, &stdout, &stderr)
			if gotCode != tc.wantCode {
				t.Fatalf("unexpected exit code: got %d want %d; stderr=%q", gotCode, tc.wantCode, stderr.String())
			}
			if tc.wantStderrPart != "" && !strings.Contains(stderr.String(), tc.wantStderrPart) {
				t.Fatalf("stderr missing %q: %q", tc.wantStderrPart, stderr.String())
			}
			if tc.wantNoWarning && strings.Contains(stderr.String(), "WARNING") {
				t.Fatalf("unexpected warning: %q", stderr.String())
			}
			if tc.wantCode != exitSuccess {
				return
			}

			loaded, err := loadATBConfig(filepath.Join(".atb", "config.json"))
			if err != nil {
				t.Fatalf("load saved config: %v", err)
			}
			if loaded.Retention == nil {
				t.Fatalf("expected retention policy")
			}
			if loaded.Retention.Days != tc.wantDays {
				t.Fatalf("unexpected saved days: got %d want %d", loaded.Retention.Days, tc.wantDays)
			}
			audit, err := bundle.LoadVerified(retentionaudit.DefaultPath())
			if err != nil {
				t.Fatalf("load retention audit: %v", err)
			}
			if got := audit.Records[len(audit.Records)-1].Event.Type; got != event.TypeDataRetentionPolicySet {
				t.Fatalf("audit event type = %q, want %q", got, event.TypeDataRetentionPolicySet)
			}
		})
	}
}

func TestConfigRetentionChangeLinksPreviousPolicyDigest(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	if code := runConfig([]string{"retention", "--days", "183"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("initial config code = %d, stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runConfig([]string{"retention", "--days", "365"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("changed config code = %d, stderr=%q", code, stderr.String())
	}
	audit, err := bundle.LoadVerified(retentionaudit.DefaultPath())
	if err != nil {
		t.Fatal(err)
	}
	last := audit.Records[len(audit.Records)-1].Event
	if last.Type != event.TypeDataRetentionPolicyChanged {
		t.Fatalf("last event = %q", last.Type)
	}
	data, _ := last.Data.(map[string]any)
	if data["previous_config_digest"] == "" || data["config_digest"] == data["previous_config_digest"] {
		t.Fatalf("policy change digests not linked correctly: %#v", data)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	now := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
	policy := defaultRetentionPolicy(90, now)
	cfg := atbConfig{
		Version:   configVersion,
		Retention: policy,
		Push: &pushSettings{
			Target:            "s3://audit-bucket/atb-prod",
			EndpointURL:       "https://storage.example.test",
			Region:            "ap-southeast-2",
			LockMode:          "COMPLIANCE",
			LockUntil:         "2028-01-01",
			CredentialsSource: "aws_env_shared",
		},
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
	if loaded.Push == nil {
		t.Fatalf("expected push settings")
	}
	if loaded.Push.Target != "s3://audit-bucket/atb-prod" {
		t.Fatalf("unexpected push target: got %q", loaded.Push.Target)
	}
	if loaded.Push.EndpointURL != "https://storage.example.test" {
		t.Fatalf("unexpected endpoint URL: got %q", loaded.Push.EndpointURL)
	}
	if loaded.Push.Region != "ap-southeast-2" {
		t.Fatalf("unexpected region: got %q", loaded.Push.Region)
	}
	if loaded.Push.LockMode != "COMPLIANCE" {
		t.Fatalf("unexpected lock mode: got %q", loaded.Push.LockMode)
	}
	if loaded.Push.LockUntil != "2028-01-01" {
		t.Fatalf("unexpected lock_until: got %q", loaded.Push.LockUntil)
	}
	if loaded.Push.CredentialsSource != "aws_env_shared" {
		t.Fatalf("unexpected credentials source: got %q", loaded.Push.CredentialsSource)
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

func TestConfigLoadPushMissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), ".atb", "config.json")
	settings, err := loadPushSettings(missingPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings != nil {
		t.Fatalf("expected nil push settings when config file is missing")
	}
}
