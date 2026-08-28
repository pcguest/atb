// SPDX-License-Identifier: MIT
package main

import (
	"archive/zip"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/trust"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestIncidentReviewWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping incident review workflow in short mode")
	}

	repoRoot := repoRootForInstalledBinarySmoke(t)
	binaryPath := buildInstalledBinary(t, repoRoot)
	workDir := t.TempDir()

	runCLI(t, binaryPath, workDir, "init")

	first := runCLIJSON[mutationResult](
		t,
		binaryPath,
		workDir,
		"append",
		"agent.run",
		"--data",
		`{"workflow":"support-triage","case_id":"case-1042","severity":"sev2"}`,
		"--format",
		"json",
	)
	if first.EventType != "agent.run" || first.Sequence != 1 {
		t.Fatalf("unexpected first incident event: %+v", first)
	}

	second := runCLIJSON[mutationResult](
		t,
		binaryPath,
		workDir,
		"append",
		"policy.alert",
		"--data",
		`{"check":"pii_redaction","outcome":"fail","ticket_id":"case-1042","reason":"customer_email_left_visible"}`,
		"--format",
		"json",
	)
	if second.EventType != "policy.alert" || second.Sequence != 2 {
		t.Fatalf("unexpected second incident event: %+v", second)
	}

	snapshot := runCLIJSON[mutationResult](
		t,
		binaryPath,
		workDir,
		"snapshot",
		"incident.review",
		"--format",
		"json",
	)
	if snapshot.EventType != event.TypeSnapshot {
		t.Fatalf("unexpected incident snapshot type: %+v", snapshot)
	}

	verifyResult := runCLIJSON[verifypkg.VerifierReport](t, binaryPath, workDir, "verify", "--format", "json")
	if verifyResult.BundlePath != bundle.DefaultPath() {
		t.Fatalf("unexpected incident verify bundle path: got %q want %q", verifyResult.BundlePath, bundle.DefaultPath())
	}
	if verifyResult.ProfileID != "" || verifyResult.Pass || len(verifyResult.Failures) != 0 {
		t.Fatalf("unexpected incident verify result: %+v", verifyResult)
	}

	trustReport := runCLIJSONExpectExitCode[verifypkg.TrustReport](t, exitUserError, binaryPath, workDir, "trust-report", "--format", "json")
	if trustReport.Pass {
		t.Fatalf("expected incident trust report to fail without a matched profile, got %+v", trustReport)
	}
	if trustReport.ProfileID != "" {
		t.Fatalf("expected no matched profile, got %+v", trustReport)
	}
	if trustReport.Chain.RecordCount != 4 {
		t.Fatalf("unexpected incident trust chain length: got %d want 4", trustReport.Chain.RecordCount)
	}

	loaded, err := bundle.Load(filepath.Join(workDir, bundle.DefaultPath()))
	if err != nil {
		t.Fatalf("load incident bundle: %v", err)
	}
	if len(loaded.Records) != 4 {
		t.Fatalf("unexpected incident bundle length: got %d want 4", len(loaded.Records))
	}
	last := loaded.Records[len(loaded.Records)-1]
	if last.Event.Type != event.TypeSnapshot {
		t.Fatalf("unexpected final incident event type: %q", last.Event.Type)
	}
	lastData, ok := last.Event.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected final incident snapshot data map, got %T", last.Event.Data)
	}
	if lastData["name"] != "incident.review" {
		t.Fatalf("expected incident snapshot name, got %v", lastData["name"])
	}

	runCLI(
		t,
		binaryPath,
		workDir,
		"export",
		"--format",
		"compliance",
		"--output",
		"incident-review-evidence.zip",
	)

	zr, err := zip.OpenReader(filepath.Join(workDir, "incident-review-evidence.zip"))
	if err != nil {
		t.Fatalf("open incident evidence zip: %v", err)
	}
	defer zr.Close()

	requiredEntries := []string{
		"evidence/docs/specification/bundle-v1.md",
		"evidence/reports/trust-report.json",
		"evidence/reports/verify.json",
	}
	for _, entry := range requiredEntries {
		if !containsString(zipNames(zr.File), entry) {
			t.Fatalf("missing incident evidence entry %q", entry)
		}
	}

	var exportedTrustReport trust.Report
	if err := json.Unmarshal(readZipFile(t, zr.File, "evidence/reports/trust-report.json"), &exportedTrustReport); err != nil {
		t.Fatalf("decode incident exported trust report: %v", err)
	}
	if exportedTrustReport.Gate.Status != trust.StatusPass {
		t.Fatalf("unexpected incident exported trust gate: %+v", exportedTrustReport.Gate)
	}
}

func zipNames(files []*zip.File) []string {
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.Name)
	}
	return names
}
