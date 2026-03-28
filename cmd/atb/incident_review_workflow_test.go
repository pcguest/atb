package main

import (
	"archive/zip"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/trust"
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
		"--gate",
		"fail",
		"--format",
		"json",
	)
	if snapshot.EventType != "snapshot.incident.review" {
		t.Fatalf("unexpected incident snapshot type: %+v", snapshot)
	}
	if snapshot.Gate != "fail" {
		t.Fatalf("unexpected incident snapshot gate: %+v", snapshot)
	}

	verifyResult := runCLIJSON[verifyResult](t, binaryPath, workDir, "verify", "--format", "json")
	if verifyResult.Status != "valid" || verifyResult.ChainLength != 4 {
		t.Fatalf("unexpected incident verify result: %+v", verifyResult)
	}

	trustReport := runCLIJSON[trust.Report](t, binaryPath, workDir, "trust-report", "--format", "json")
	if trustReport.Status != trust.StatusPass {
		t.Fatalf("expected incident trust report to pass, got %+v", trustReport)
	}
	if trustReport.Gate.Status != trust.StatusPass {
		t.Fatalf("expected incident trust gate to pass, got %+v", trustReport.Gate)
	}
	if trustReport.ChainLength != 4 {
		t.Fatalf("unexpected incident trust chain length: got %d want 4", trustReport.ChainLength)
	}

	loaded, err := bundle.Load(filepath.Join(workDir, bundle.DefaultPath()))
	if err != nil {
		t.Fatalf("load incident bundle: %v", err)
	}
	if len(loaded.Records) != 4 {
		t.Fatalf("unexpected incident bundle length: got %d want 4", len(loaded.Records))
	}
	last := loaded.Records[len(loaded.Records)-1]
	if last.Event.Type != "snapshot.incident.review" {
		t.Fatalf("unexpected final incident event type: %q", last.Event.Type)
	}
	lastData, ok := last.Event.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected final incident snapshot data map, got %T", last.Event.Data)
	}
	if lastData["gate"] != "fail" {
		t.Fatalf("expected incident snapshot gate fail, got %v", lastData["gate"])
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
		"evidence/docs/spec-v1.0.md",
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
