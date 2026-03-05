package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	archiveledger "github.com/pcguest/atb/internal/archive"
	"github.com/pcguest/atb/internal/bundle"
)

func TestExportParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    exportConfig
		wantErr bool
	}{
		{
			name: "full config",
			args: []string{"--format", "compliance", "--output", "evidence.zip", "--dry-run"},
			want: exportConfig{Format: "compliance", Output: "evidence.zip", DryRun: true},
		},
		{
			name: "equals syntax",
			args: []string{"--format=compliance", "--output=out.zip"},
			want: exportConfig{Format: "compliance", Output: "out.zip", DryRun: false},
		},
		{
			name:    "missing output",
			args:    []string{"--format", "compliance"},
			wantErr: true,
		},
		{
			name:    "invalid format",
			args:    []string{"--format", "json", "--output", "out.zip"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--output", "out.zip", "--wat"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseExportArgs(tc.args)
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

func TestExportDryRunBuildsWithoutWritingZip(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		writeValidBundle(t, filepath.Join("run.atb", "bundle.atb"))
		cfg := exportConfig{Format: exportFormatCompliance, Output: "evidence.zip", DryRun: true}

		result, err := buildComplianceExport(time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), cfg)
		if err != nil {
			t.Fatalf("build compliance export: %v", err)
		}
		if result.Manifest.Verification.ActiveVerified != 1 {
			t.Fatalf("unexpected active verification count: got %d want 1", result.Manifest.Verification.ActiveVerified)
		}
		if _, err := os.Stat(filepath.Join(tmp, "evidence.zip")); !os.IsNotExist(err) {
			t.Fatalf("dry-run should not create zip output")
		}
	})
}

func TestExportFailsOnVerificationError(t *testing.T) {
	withTempCWD(t, func(_ string) {
		writeValidBundle(t, filepath.Join("run.atb", "bundle.atb"))
		if err := os.WriteFile(filepath.Join("run.atb", "bad.atb"), []byte("not-ndjson\n"), 0600); err != nil {
			t.Fatalf("write invalid bundle: %v", err)
		}

		cfg := exportConfig{Format: exportFormatCompliance, Output: "evidence.zip"}
		_, err := buildComplianceExport(time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), cfg)
		if err == nil {
			t.Fatalf("expected verification error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "verification") {
			t.Fatalf("expected verification error, got %v", err)
		}
	})
}

func TestExportComplianceZipStructure(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		writeValidBundle(t, filepath.Join("run.atb", "bundle.atb"))

		archivedPath := filepath.Join("archive.atb", "2026", "03", "05", "run.atb", "legacy.atb")
		writeValidBundle(t, archivedPath)
		archivedSHA, err := computeFileSHA256(archivedPath)
		if err != nil {
			t.Fatalf("sha archived: %v", err)
		}
		archivedBundle, err := bundle.Load(archivedPath)
		if err != nil {
			t.Fatalf("load archived bundle: %v", err)
		}
		archivedHead := archivedBundle.Records[len(archivedBundle.Records)-1].Hash

		ledgerPath := filepath.Join("archive.atb", archiveledger.LedgerFile)
		entries, err := archiveledger.LoadOrEmpty(ledgerPath)
		if err != nil {
			t.Fatalf("load empty ledger: %v", err)
		}
		entry, err := archiveledger.NextEntry(entries, "2026-03-05T12:00:00Z", "run.atb/legacy.atb", filepath.ToSlash(archivedPath), archivedSHA, archivedHead)
		if err != nil {
			t.Fatalf("create ledger entry: %v", err)
		}
		if err := archiveledger.Append(ledgerPath, entry); err != nil {
			t.Fatalf("append ledger entry: %v", err)
		}

		cfgModel := atbConfig{Version: configVersion, Retention: defaultRetentionPolicy(90, time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC))}
		if err := saveATBConfig(defaultConfigPath(), cfgModel); err != nil {
			t.Fatalf("save config: %v", err)
		}

		if err := os.MkdirAll("docs", 0750); err != nil {
			t.Fatalf("mkdir docs: %v", err)
		}
		mustWrite := func(path, content string) {
			t.Helper()
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
		}
		mustWrite("docs/spec-v1.0.md", "spec")
		mustWrite("docs/security.md", "security")
		mustWrite("docs/incident-response.md", "incident")
		if err := os.MkdirAll("docs/compliance", 0750); err != nil {
			t.Fatalf("mkdir docs/compliance: %v", err)
		}
		mustWrite("docs/compliance/export.md", "export")

		cfg := exportConfig{Format: exportFormatCompliance, Output: "evidence.zip"}
		result, err := buildComplianceExport(time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), cfg)
		if err != nil {
			t.Fatalf("build export: %v", err)
		}
		if err := writeComplianceZip(result); err != nil {
			t.Fatalf("write zip: %v", err)
		}

		zr, err := zip.OpenReader(filepath.Join(tmp, "evidence.zip"))
		if err != nil {
			t.Fatalf("open zip: %v", err)
		}
		defer zr.Close()

		names := make([]string, 0, len(zr.File))
		for _, f := range zr.File {
			names = append(names, f.Name)
		}
		sort.Strings(names)

		required := []string{
			"evidence/checksums.chain",
			"evidence/checksums.sha256",
			"evidence/config/atb-config.json",
			"evidence/docs/compliance/export.md",
			"evidence/docs/incident-response.md",
			"evidence/docs/security.md",
			"evidence/docs/spec-v1.0.md",
			"evidence/manifest.json",
			"evidence/reports/archive-ledger.json",
			"evidence/reports/trust-report.json",
			"evidence/reports/verify.json",
		}
		for _, want := range required {
			if !containsString(names, want) {
				t.Fatalf("missing required zip entry %q", want)
			}
		}

		manifestBytes := readZipFile(t, zr.File, "evidence/manifest.json")
		var manifest exportManifest
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}
		if manifest.Verification.ActiveVerified != 1 {
			t.Fatalf("unexpected active_verified: got %d want 1", manifest.Verification.ActiveVerified)
		}
		if manifest.Verification.ArchivedVerified != 1 {
			t.Fatalf("unexpected archived_verified: got %d want 1", manifest.Verification.ArchivedVerified)
		}
		if !manifest.Verification.LedgerVerified {
			t.Fatalf("expected ledger_verified true")
		}

		checksums := string(readZipFile(t, zr.File, "evidence/checksums.sha256"))
		if strings.Contains(checksums, "chain:") {
			t.Fatalf("expected checksums.sha256 to remain sha256sum-compatible")
		}
		meta := string(readZipFile(t, zr.File, "evidence/checksums.chain"))
		if !strings.Contains(meta, "chain:") {
			t.Fatalf("expected checksums.chain to include chain lines")
		}
		if !strings.Contains(meta, "file:") {
			t.Fatalf("expected checksums.chain to include file annotation lines")
		}
	})
}

func containsString(xs []string, needle string) bool {
	for _, x := range xs {
		if x == needle {
			return true
		}
	}
	return false
}

func readZipFile(t *testing.T, files []*zip.File, name string) []byte {
	t.Helper()
	for _, f := range files {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", name, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read zip entry %s: %v", name, err)
		}
		return data
	}
	t.Fatalf("zip entry not found: %s", name)
	return nil
}
