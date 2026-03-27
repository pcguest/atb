package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	archiveledger "github.com/pcguest/atb/internal/archive"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
)

func TestExportParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    exportConfig
		wantErr bool
	}{
		{
			name: "compliance config",
			args: []string{"--format", "compliance", "--output", "evidence.zip", "--dry-run"},
			want: exportConfig{Format: exportFormatCompliance, Output: "evidence.zip", DryRun: true},
		},
		{
			name: "soc2 with bundle",
			args: []string{"--format=soc2", "--bundle", "run.atb/bundle.atb", "--output=soc2.zip"},
			want: exportConfig{Format: exportFormatSOC2, BundlePath: filepath.Join("run.atb", "bundle.atb"), Output: "soc2.zip"},
		},
		{
			name: "gdpr dsr",
			args: []string{"--format", "gdpr", "--type", "dsr", "--subject-id", "usr_9f8e7d6c", "--bundle", "run.atb/bundle.atb", "--output", "gdpr.zip"},
			want: exportConfig{Format: exportFormatGDPR, GDPRType: exportGDPRTypeDSR, SubjectID: "usr_9f8e7d6c", BundlePath: filepath.Join("run.atb", "bundle.atb"), Output: "gdpr.zip"},
		},
		{
			name: "gdpr ropa defaults bundle",
			args: []string{"--format", "gdpr", "--type=ropa", "--output", "gdpr.zip"},
			want: exportConfig{Format: exportFormatGDPR, GDPRType: exportGDPRTypeROPA, BundlePath: bundle.DefaultPath(), Output: "gdpr.zip"},
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
			name:    "gdpr missing type",
			args:    []string{"--format", "gdpr", "--output", "out.zip"},
			wantErr: true,
		},
		{
			name:    "gdpr dsr missing subject",
			args:    []string{"--format", "gdpr", "--type", "dsr", "--output", "out.zip"},
			wantErr: true,
		},
		{
			name:    "compliance with gdpr flags",
			args:    []string{"--format", "compliance", "--type", "ropa", "--output", "out.zip"},
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

		result, err := buildExport(time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), cfg)
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

func TestExportUsesEmbeddedDocsOutsideRepoCheckout(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		writeValidBundle(t, filepath.Join("run.atb", "bundle.atb"))

		now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
		cfg := exportConfig{Format: exportFormatSOC2, BundlePath: bundle.DefaultPath(), Output: "soc2.zip"}
		result, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build export with embedded docs: %v", err)
		}
		if err := writeExportZip(result, now); err != nil {
			t.Fatalf("write export zip: %v", err)
		}

		zr, err := zip.OpenReader(filepath.Join(tmp, "soc2.zip"))
		if err != nil {
			t.Fatalf("open zip: %v", err)
		}
		defer zr.Close()

		required := []string{
			"evidence/docs/compliance/soc2.md",
			"evidence/docs/incident-response.md",
			"evidence/docs/security.md",
			"evidence/docs/spec-v1.0.md",
		}
		names := make([]string, 0, len(zr.File))
		for _, f := range zr.File {
			names = append(names, f.Name)
		}
		for _, want := range required {
			if !containsString(names, want) {
				t.Fatalf("missing embedded doc %q", want)
			}
		}
	})
}

func TestExportFailsOnVerificationError(t *testing.T) {
	withTempCWD(t, func(_ string) {
		writeValidBundle(t, filepath.Join("run.atb", "bundle.atb"))
		prepareComplianceDocs(t)
		if err := os.WriteFile(filepath.Join("run.atb", "bad.atb"), []byte("not-ndjson\n"), 0600); err != nil {
			t.Fatalf("write invalid bundle: %v", err)
		}

		cfg := exportConfig{Format: exportFormatCompliance, Output: "evidence.zip"}
		_, err := buildExport(time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), cfg)
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
		prepareComplianceDocs(t)

		now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
		cfg := exportConfig{Format: exportFormatCompliance, Output: "evidence.zip"}
		result, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build export: %v", err)
		}
		if err := writeExportZip(result, now); err != nil {
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
	})
}

func TestExportSOC2SchemaMatchesGolden(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		preparePhase4Docs(t)
		writePhase4Bundle(t, bundle.DefaultPath())

		now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
		cfg := exportConfig{Format: exportFormatSOC2, BundlePath: bundle.DefaultPath(), Output: "soc2.zip"}
		result, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build soc2 export: %v", err)
		}
		if err := writeExportZip(result, now); err != nil {
			t.Fatalf("write soc2 zip: %v", err)
		}

		zr, err := zip.OpenReader(filepath.Join(tmp, "soc2.zip"))
		if err != nil {
			t.Fatalf("open zip: %v", err)
		}
		defer zr.Close()

		assertJSONMatchesFixture(t, readZipFile(t, zr.File, "evidence/soc2_evidence_manifest.json"), "export_soc2_manifest.json")
		if len(readZipFile(t, zr.File, "evidence/audit_trail.jsonl")) == 0 {
			t.Fatalf("expected non-empty audit_trail.jsonl")
		}
		if len(readZipFile(t, zr.File, "evidence/verification_report.json")) == 0 {
			t.Fatalf("expected non-empty verification_report.json")
		}
	})
}

func TestExportGDPRDSRSchemaMatchesGolden(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		preparePhase4Docs(t)
		writePhase4Bundle(t, bundle.DefaultPath())

		now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
		cfg := exportConfig{
			Format:     exportFormatGDPR,
			GDPRType:   exportGDPRTypeDSR,
			SubjectID:  "usr_9f8e7d6c",
			BundlePath: bundle.DefaultPath(),
			Output:     "gdpr-dsr.zip",
		}
		result, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build gdpr dsr export: %v", err)
		}
		if err := writeExportZip(result, now); err != nil {
			t.Fatalf("write gdpr dsr zip: %v", err)
		}

		zr, err := zip.OpenReader(filepath.Join(tmp, "gdpr-dsr.zip"))
		if err != nil {
			t.Fatalf("open zip: %v", err)
		}
		defer zr.Close()

		assertJSONMatchesFixture(t, readZipFile(t, zr.File, "evidence/dsr_usr_9f8e7d6c.json"), "export_gdpr_dsr.json")
	})
}

func TestExportGDPRROPASchemaMatchesGolden(t *testing.T) {
	withTempCWD(t, func(tmp string) {
		preparePhase4Docs(t)
		writePhase4Bundle(t, bundle.DefaultPath())

		now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
		cfg := exportConfig{
			Format:     exportFormatGDPR,
			GDPRType:   exportGDPRTypeROPA,
			BundlePath: bundle.DefaultPath(),
			Output:     "gdpr-ropa.zip",
		}
		result, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build gdpr ropa export: %v", err)
		}
		if err := writeExportZip(result, now); err != nil {
			t.Fatalf("write gdpr ropa zip: %v", err)
		}

		zr, err := zip.OpenReader(filepath.Join(tmp, "gdpr-ropa.zip"))
		if err != nil {
			t.Fatalf("open zip: %v", err)
		}
		defer zr.Close()

		assertJSONMatchesFixture(t, readZipFile(t, zr.File, "evidence/ropa_summary.json"), "export_gdpr_ropa.json")
	})
}

func TestExportGDPRDSRSubjectMissingFails(t *testing.T) {
	withTempCWD(t, func(_ string) {
		preparePhase4Docs(t)
		writePhase4Bundle(t, bundle.DefaultPath())

		cfg := exportConfig{
			Format:     exportFormatGDPR,
			GDPRType:   exportGDPRTypeDSR,
			SubjectID:  "usr_missing",
			BundlePath: bundle.DefaultPath(),
			Output:     "gdpr.zip",
		}
		_, err := buildExport(time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC), cfg)
		if err == nil {
			t.Fatalf("expected subject not found error")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected subject error, got %v", err)
		}
	})
}

func TestExportGDPRRetentionExpiredFails(t *testing.T) {
	withTempCWD(t, func(_ string) {
		preparePhase4Docs(t)
		writePhase4Bundle(t, bundle.DefaultPath())

		cfgModel := atbConfig{Version: configVersion, Retention: defaultRetentionPolicy(1, time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC))}
		if err := saveATBConfig(defaultConfigPath(), cfgModel); err != nil {
			t.Fatalf("save config: %v", err)
		}

		now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
		old := now.AddDate(0, 0, -3)
		if err := os.Chtimes(bundle.DefaultPath(), old, old); err != nil {
			t.Fatalf("set old mtime: %v", err)
		}

		cfg := exportConfig{
			Format:     exportFormatGDPR,
			GDPRType:   exportGDPRTypeROPA,
			BundlePath: bundle.DefaultPath(),
			Output:     "gdpr.zip",
		}
		_, err := buildExport(now, cfg)
		if err == nil {
			t.Fatalf("expected retention expiry error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "retention") {
			t.Fatalf("expected retention error, got %v", err)
		}
	})
}

func TestExportDeterministicZipBytesWithFixedTime(t *testing.T) {
	withTempCWD(t, func(_ string) {
		preparePhase4Docs(t)
		writePhase4Bundle(t, bundle.DefaultPath())

		now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
		cfg := exportConfig{Format: exportFormatSOC2, BundlePath: bundle.DefaultPath(), Output: "a.zip"}
		resultA, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build export a: %v", err)
		}
		if err := writeExportZip(resultA, now); err != nil {
			t.Fatalf("write export a: %v", err)
		}

		cfg.Output = "b.zip"
		resultB, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build export b: %v", err)
		}
		if err := writeExportZip(resultB, now); err != nil {
			t.Fatalf("write export b: %v", err)
		}

		bytesA, err := os.ReadFile("a.zip")
		if err != nil {
			t.Fatalf("read a.zip: %v", err)
		}
		bytesB, err := os.ReadFile("b.zip")
		if err != nil {
			t.Fatalf("read b.zip: %v", err)
		}
		if string(bytesA) != string(bytesB) {
			t.Fatalf("expected byte-identical zip output for fixed input/time")
		}
	})
}

func TestBuildVerifyReportUsesInjectedTime(t *testing.T) {
	withTempCWD(t, func(_ string) {
		writeValidBundle(t, bundle.DefaultPath())
		now := time.Date(2026, 3, 7, 15, 30, 0, 0, time.UTC)
		data, err := buildVerifyReport(now, []string{bundle.DefaultPath()}, nil)
		if err != nil {
			t.Fatalf("build verify report: %v", err)
		}
		var report struct {
			GeneratedAt string `json:"generated_at"`
		}
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatalf("unmarshal report: %v", err)
		}
		if report.GeneratedAt != now.Format(time.RFC3339) {
			t.Fatalf("unexpected generated_at: got %q want %q", report.GeneratedAt, now.Format(time.RFC3339))
		}
	})
}

func prepareComplianceDocs(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll("docs/compliance", 0750); err != nil {
		t.Fatalf("mkdir docs/compliance: %v", err)
	}
	mustWrite(t, "docs/spec-v1.0.md", "spec")
	mustWrite(t, "docs/security.md", "security")
	mustWrite(t, "docs/incident-response.md", "incident")
	mustWrite(t, "docs/compliance/export.md", "export")
}

func preparePhase4Docs(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll("docs/compliance", 0750); err != nil {
		t.Fatalf("mkdir docs/compliance: %v", err)
	}
	mustWrite(t, "docs/spec-v1.0.md", "spec")
	mustWrite(t, "docs/security.md", "security")
	mustWrite(t, "docs/incident-response.md", "incident")
	mustWrite(t, "docs/compliance/soc2.md", "soc2")
	mustWrite(t, "docs/compliance/gdpr.md", "gdpr")
}

func writePhase4Bundle(t *testing.T, path string) {
	t.Helper()
	b := bundle.New()

	subject := "usr_9f8e7d6c"
	org := "org_xyz"
	workspace := "ws_main"
	actor := "actor_bot"

	appendWith := func(eventType string, data map[string]interface{}, opts *bundle.AppendOptions) {
		t.Helper()
		if err := b.AppendWithOptions(eventType, data, opts); err != nil {
			t.Fatalf("append %s: %v", eventType, err)
		}
	}

	appendWith("auth.login", map[string]interface{}{
		"event_id":   "evt_001",
		"timestamp":  "2026-01-10T09:00:00Z",
		"user_id":    subject,
		"ip":         "192.168.1.1",
		"user_agent": "Mozilla/5.0",
	}, &bundle.AppendOptions{ActorID: &actor, OrgID: &org, WorkspaceID: &workspace})

	appendWith("system.config.change", map[string]interface{}{
		"event_id":    "evt_002",
		"timestamp":   "2026-01-11T10:00:00Z",
		"before_hash": "abc",
		"after_hash":  "def",
		"owner_id":    "usr_admin",
	}, &bundle.AppendOptions{ActorID: &actor, OrgID: &org, WorkspaceID: &workspace})

	appendWith("alert.triggered", map[string]interface{}{
		"event_id":  "evt_003",
		"timestamp": "2026-01-12T11:00:00Z",
		"payment":   "4111111111111111",
		"ip":        "10.0.0.2",
		"user_id":   "usr_other",
	}, &bundle.AppendOptions{ActorID: &actor, OrgID: &org, WorkspaceID: &workspace})

	appendWith("deploy.complete", map[string]interface{}{
		"event_id":    "evt_004",
		"timestamp":   "2026-01-13T12:00:00Z",
		"commit_hash": "deadbeef",
		"approver_id": "usr_admin",
		"ci_run_id":   "ci_1",
	}, &bundle.AppendOptions{ActorID: &actor, OrgID: &org, WorkspaceID: &workspace})

	appendWith("backup.complete", map[string]interface{}{
		"event_id":            "evt_005",
		"timestamp":           "2026-01-14T13:00:00Z",
		"storage_location_id": "s3://bucket",
		"verification_status": "ok",
	}, &bundle.AppendOptions{ActorID: &actor, OrgID: &org, WorkspaceID: &workspace})

	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertJSONMatchesFixture(t *testing.T, got []byte, fixtureName string) {
	t.Helper()
	want, err := os.ReadFile(fixturePath(t, fixtureName))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixtureName, err)
	}
	gotCanonical, err := canonicalize.MarshalRaw(got)
	if err != nil {
		t.Fatalf("canonicalize got %s: %v", fixtureName, err)
	}
	wantCanonical, err := canonicalize.MarshalRaw(want)
	if err != nil {
		t.Fatalf("canonicalize want %s: %v", fixtureName, err)
	}
	if string(gotCanonical) != string(wantCanonical) {
		t.Fatalf("fixture mismatch for %s\n got: %s\nwant: %s", fixtureName, string(got), string(want))
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(repoRoot, "test", "golden", name)
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
