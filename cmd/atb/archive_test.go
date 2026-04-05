package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	archiveledger "github.com/pcguest/atb/internal/archive"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func TestArchiveParseArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantDry    bool
		wantBefore string
		wantErr    bool
	}{
		{
			name:       "before split with dry run",
			args:       []string{"--before", "2025-01-01", "--dry-run"},
			wantDry:    true,
			wantBefore: "2025-01-01",
		},
		{
			name:       "before equals",
			args:       []string{"--before=2025-01-01"},
			wantBefore: "2025-01-01",
		},
		{
			name:    "invalid date",
			args:    []string{"--before", "2025/01/01"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--wat"},
			wantErr: true,
		},
		{
			name:    "unexpected positional",
			args:    []string{"run.atb/bundle.atb"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArchiveArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.DryRun != tc.wantDry {
				t.Fatalf("unexpected dry_run value: got %v want %v", got.DryRun, tc.wantDry)
			}
			if tc.wantBefore != "" {
				if !got.BeforeSet {
					t.Fatalf("expected before to be set")
				}
				if got.Before.Format("2006-01-02") != tc.wantBefore {
					t.Fatalf("unexpected before date: got %s want %s", got.Before.Format("2006-01-02"), tc.wantBefore)
				}
			}
		})
	}
}

func TestArchiveDryRunNoSideEffects(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		sourcePath := filepath.Join("run.atb", "old.atb")
		writeValidBundle(t, sourcePath)
		setFileTime(t, sourcePath, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))

		cfg := archiveConfig{
			BeforeSet: true,
			Before:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			DryRun:    true,
		}
		now := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)

		var out bytes.Buffer
		summary, err := runArchive(now, cfg, &out)
		if err != nil {
			t.Fatalf("run archive dry-run: %v", err)
		}
		if summary.WouldArchive != 1 {
			t.Fatalf("unexpected would_archive count: got %d want 1", summary.WouldArchive)
		}
		if summary.Archived != 0 {
			t.Fatalf("unexpected archived count in dry-run: got %d want 0", summary.Archived)
		}

		if _, err := os.Stat(sourcePath); err != nil {
			t.Fatalf("expected source bundle to remain: %v", err)
		}
		destPath := filepath.Join("archive.atb", "2026", "03", "05", "run.atb", "old.atb")
		if _, err := os.Stat(destPath); !os.IsNotExist(err) {
			t.Fatalf("expected destination bundle to be absent in dry-run")
		}
		ledgerPath := filepath.Join("archive.atb", archiveledger.LedgerFile)
		if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
			t.Fatalf("expected ledger file to be absent in dry-run")
		}

		output := out.String()
		if !strings.Contains(output, "~ would archive") {
			t.Fatalf("expected dry-run output to mention would archive, got: %s", output)
		}
	})
}

func TestArchiveSkipsInvalidBundle(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		sourcePath := filepath.Join("run.atb", "invalid.atb")
		if err := os.MkdirAll(filepath.Dir(sourcePath), 0750); err != nil {
			t.Fatalf("mkdir run dir: %v", err)
		}
		if err := os.WriteFile(sourcePath, []byte("not-ndjson\n"), 0600); err != nil {
			t.Fatalf("write invalid bundle: %v", err)
		}
		setFileTime(t, sourcePath, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))

		cfg := archiveConfig{
			BeforeSet: true,
			Before:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		now := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)

		var out bytes.Buffer
		summary, err := runArchive(now, cfg, &out)
		if err != nil {
			t.Fatalf("run archive for invalid bundle: %v", err)
		}
		if summary.SkippedInvalid != 1 {
			t.Fatalf("unexpected skipped_invalid count: got %d want 1", summary.SkippedInvalid)
		}
		if summary.Archived != 0 {
			t.Fatalf("unexpected archived count: got %d want 0", summary.Archived)
		}
		if _, err := os.Stat(sourcePath); err != nil {
			t.Fatalf("expected invalid source bundle to remain: %v", err)
		}
		ledgerPath := filepath.Join("archive.atb", archiveledger.LedgerFile)
		if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
			t.Fatalf("expected no ledger file when nothing archived")
		}
	})
}

func TestArchiveRecreatesDefaultBundleAndNormalizesLedgerPaths(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		sourcePath := bundle.DefaultPath()
		writeValidBundle(t, sourcePath)
		setFileTime(t, sourcePath, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))

		cfg := archiveConfig{
			BeforeSet: true,
			Before:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		now := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)

		var out bytes.Buffer
		summary, err := runArchive(now, cfg, &out)
		if err != nil {
			t.Fatalf("run archive: %v", err)
		}
		if summary.Archived != 1 {
			t.Fatalf("unexpected archived count: got %d want 1", summary.Archived)
		}
		if !summary.ArchivedDefault {
			t.Fatalf("expected default bundle recreation flag")
		}

		defaultBundle, err := bundle.Load(bundle.DefaultPath())
		if err != nil {
			t.Fatalf("load recreated default bundle: %v", err)
		}
		if len(defaultBundle.Records) != 1 {
			t.Fatalf("expected recreated default bundle to contain only the manifest, got %d records", len(defaultBundle.Records))
		}
		if defaultBundle.Manifest() == nil {
			t.Fatalf("expected recreated default bundle manifest")
		}

		destPath := filepath.Join("archive.atb", "2026", "03", "05", "run.atb", "bundle.atb")
		if _, err := os.Stat(destPath); err != nil {
			t.Fatalf("expected archived bundle at destination: %v", err)
		}

		ledgerPath := filepath.Join("archive.atb", archiveledger.LedgerFile)
		entries, err := archiveledger.Load(ledgerPath)
		if err != nil {
			t.Fatalf("load ledger: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("unexpected ledger entry count: got %d want 1", len(entries))
		}
		entry := entries[0]
		if strings.Contains(entry.Source, "\\") {
			t.Fatalf("expected slash-normalized source path, got %q", entry.Source)
		}
		if strings.Contains(entry.Dest, "\\") {
			t.Fatalf("expected slash-normalized destination path, got %q", entry.Dest)
		}
		if entry.Source != "run.atb/bundle.atb" {
			t.Fatalf("unexpected source path in ledger: got %q", entry.Source)
		}
		expectedDest := filepath.ToSlash(filepath.Join("archive.atb", "2026", "03", "05", "run.atb", "bundle.atb"))
		if entry.Dest != expectedDest {
			t.Fatalf("unexpected destination path in ledger: got %q want %q", entry.Dest, expectedDest)
		}

		head, err := archiveledger.Verify(entries)
		if err != nil {
			t.Fatalf("verify ledger chain: %v", err)
		}
		if head == hash.GenesisHash {
			t.Fatalf("expected non-genesis ledger head hash")
		}
	})
}

func TestArchiveUsesRetentionConfigWhenBeforeUnset(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		cfgPath := defaultConfigPath()
		cfg := atbConfig{Version: configVersion, Retention: defaultRetentionPolicy(90, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))}
		if err := saveATBConfig(cfgPath, cfg); err != nil {
			t.Fatalf("save config: %v", err)
		}

		sourcePath := filepath.Join("run.atb", "old.atb")
		writeValidBundle(t, sourcePath)
		setFileTime(t, sourcePath, time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC))

		now := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
		var out bytes.Buffer
		summary, err := runArchive(now, archiveConfig{}, &out)
		if err != nil {
			t.Fatalf("run archive with retention config: %v", err)
		}
		if summary.Archived != 1 {
			t.Fatalf("unexpected archived count using retention policy: got %d want 1", summary.Archived)
		}
	})
}

func withTempCWD(t *testing.T, fn func(tmp string)) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()
	fn(tmp)
}

func writeValidBundle(t *testing.T, path string) {
	t.Helper()
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "test.event", map[string]interface{}{"ok": true})
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
}

func setFileTime(t *testing.T, path string, ts time.Time) {
	t.Helper()
	if err := os.Chtimes(path, ts, ts); err != nil {
		t.Fatalf("set file time: %v", err)
	}
}
