// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/event"
	exportpkg "github.com/pcguest/atb/internal/export"
)

func TestExportComplianceManifestGolden(t *testing.T) {
	bundlePath := buildSnapshotBundle(t, []snapshotAppend{
		{
			eventType: event.TypeAIRequestReceived,
			data: map[string]any{
				"request_id":    "req-001",
				"actor_id_hash": "sha256-actor",
				"purpose_tag":   "rag_answer",
			},
		},
		{
			eventType: event.TypeAIModelInvoked,
			data: map[string]any{
				"model_provider":          "openai",
				"model_id":                "gpt-4o",
				"model_parameters_digest": "sha256-params",
				"prompt_digest":           "sha256-prompt",
			},
		},
		{
			eventType: event.TypeAIModelOutput,
			data: map[string]any{
				"output_digest": "sha256-output",
				"output_format": "text/plain",
			},
		},
	})

	stdout := captureComplianceManifestJSON(t, bundlePath)

	var manifest exportpkg.ComplianceManifest
	if err := json.Unmarshal(stdout, &manifest); err != nil {
		t.Fatalf("unmarshal compliance manifest: %v\noutput=%s", err, string(stdout))
	}

	if manifest.ExportFormat != exportFormatCompliance {
		t.Fatalf("ExportFormat = %q, want %q", manifest.ExportFormat, exportFormatCompliance)
	}
	if manifest.VerifyResult == nil {
		t.Fatalf("VerifyResult is nil")
	}
	if manifest.VerifyResult.ProfileID != "atb.profile.rag_answer" {
		t.Fatalf("VerifyResult.ProfileID = %q, want %q", manifest.VerifyResult.ProfileID, "atb.profile.rag_answer")
	}
	if !manifest.VerifyResult.Pass {
		t.Fatalf("VerifyResult.Pass = false, want true")
	}
	if len(manifest.Files) == 0 {
		t.Fatalf("Files is empty")
	}
	if got, want := len(manifest.RegulatoryCoverage), 4; got != want {
		t.Fatalf("len(RegulatoryCoverage) = %d, want %d", got, want)
	}
	if !containsString(manifest.RegulatoryCoverage, "EU AI Act Article 12") {
		t.Fatalf("RegulatoryCoverage missing %q: %v", "EU AI Act Article 12", manifest.RegulatoryCoverage)
	}

	normalised := normaliseComplianceManifest(manifest)
	got, err := json.MarshalIndent(normalised, "", "  ")
	if err != nil {
		t.Fatalf("marshal normalised compliance manifest: %v", err)
	}
	got = canonicalIndentedJSON(t, got)

	goldenPath := exportComplianceManifestGoldenPath(t)
	if err := os.MkdirAll(filepath.Dir(goldenPath), 0750); err != nil {
		t.Fatalf("mkdir golden dir: %v", err)
	}
	if os.Getenv("ATB_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, got, 0600); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
	}

	if _, err := os.Stat(goldenPath); err != nil {
		if os.IsNotExist(err) {
			if writeErr := os.WriteFile(goldenPath, got, 0600); writeErr != nil {
				t.Fatalf("write golden file: %v", writeErr)
			}
			t.Logf("wrote golden file")
			return
		}
		t.Fatalf("stat golden file: %v", err)
	}

	wantRaw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	want := canonicalIndentedJSON(t, wantRaw)

	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch for %s\n%s", goldenPath, diffJSONLines(want, got))
	}
}

func captureComplianceManifestJSON(t *testing.T, bundlePath string) []byte {
	t.Helper()

	workDir := filepath.Dir(filepath.Dir(bundlePath))
	cfg := exportConfig{
		Format:     exportFormatCompliance,
		Output:     "compliance.zip",
		JSON:       true,
		BundlePath: bundlePath,
	}
	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	var stdout []byte
	withSnapshotWorkingDir(t, workDir, func() {
		result, err := buildExport(now, cfg)
		if err != nil {
			t.Fatalf("build export: %v", err)
		}

		manifest, exitCode, err := buildComplianceJSONManifest(now, cfg, result)
		if err != nil {
			t.Fatalf("build compliance json manifest: %v", err)
		}
		if exitCode != exitSuccess {
			t.Fatalf("buildComplianceJSONManifest() exit code = %d, want %d", exitCode, exitSuccess)
		}

		stdout, err = json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatalf("marshal compliance manifest: %v", err)
		}
	})

	return stdout
}

func normaliseComplianceManifest(manifest exportpkg.ComplianceManifest) exportpkg.ComplianceManifest {
	manifest.GeneratedAt = "NORMALISED"
	manifest.BundlePath = "NORMALISED"

	if manifest.VerifyResult != nil {
		manifest.VerifyResult.BundlePath = "NORMALISED"
		if manifest.VerifyResult.SubScores != nil {
			rounded := make(map[string]float64, len(manifest.VerifyResult.SubScores))
			for key, value := range manifest.VerifyResult.SubScores {
				rounded[key] = roundTo4(value)
			}
			manifest.VerifyResult.SubScores = rounded
		}
	}

	for i := range manifest.Files {
		manifest.Files[i].SizeBytes = 0
	}

	return manifest
}

func roundTo4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func exportComplianceManifestGoldenPath(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "export_compliance_manifest.golden.json")
}

func canonicalIndentedJSON(t *testing.T, data []byte) []byte {
	t.Helper()

	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("canonicalise json: %v\ninput=%s", err, string(data))
	}
	canonical, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("marshal canonical json: %v", err)
	}
	return canonical
}

func diffJSONLines(want, got []byte) string {
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")

	maxLines := len(wantLines)
	if len(gotLines) > maxLines {
		maxLines = len(gotLines)
	}

	var b strings.Builder
	b.WriteString("first differing lines:\n")
	diffCount := 0
	for i := 0; i < maxLines && diffCount < 20; i++ {
		var wantLine string
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}

		var gotLine string
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}

		if wantLine == gotLine {
			continue
		}

		diffCount++
		b.WriteString(fmt.Sprintf("line %d:\n", i+1))
		b.WriteString(fmt.Sprintf("  want: %q\n", wantLine))
		b.WriteString(fmt.Sprintf("  got:  %q\n", gotLine))
	}

	if diffCount == 0 {
		b.WriteString("no differing lines found")
	}

	return b.String()
}
