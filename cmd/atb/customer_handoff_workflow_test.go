package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/trust"
)

func TestCustomerHandoffWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping customer handoff workflow in short mode")
	}

	repoRoot := repoRootForInstalledBinarySmoke(t)
	binaryPath := buildInstalledBinary(t, repoRoot)
	senderDir := t.TempDir()
	recipientDir := t.TempDir()
	passwordEnv := []string{"ATB_PASSWORD=shared-review-secret"}

	runCLI(t, binaryPath, senderDir, "init")

	first := runCLIJSON[mutationResult](
		t,
		binaryPath,
		senderDir,
		"append",
		"agent.run",
		"--data",
		`{"workflow":"claims-triage","customer":"acme","ticket_id":"handoff-204"}`,
		"--format",
		"json",
	)
	if first.EventType != "agent.run" || first.Sequence != 1 {
		t.Fatalf("unexpected first handoff event: %+v", first)
	}

	second := runCLIJSON[mutationResult](
		t,
		binaryPath,
		senderDir,
		"append",
		"decision",
		"--data",
		`{"action":"route_to_manual_review","reason":"confidence_below_threshold","customer":"acme"}`,
		"--format",
		"json",
	)
	if second.EventType != "decision" || second.Sequence != 2 {
		t.Fatalf("unexpected second handoff event: %+v", second)
	}

	snapshot := runCLIJSON[mutationResult](
		t,
		binaryPath,
		senderDir,
		"snapshot",
		"customer_handoff",
		"--gate",
		"pass",
		"--format",
		"json",
	)
	if snapshot.EventType != "snapshot.customer_handoff" || snapshot.Gate != "pass" {
		t.Fatalf("unexpected handoff snapshot: %+v", snapshot)
	}

	senderVerify := runCLIJSON[verifyResult](t, binaryPath, senderDir, "verify", "--format", "json")
	if senderVerify.Status != "valid" || senderVerify.ChainLength != 3 {
		t.Fatalf("unexpected sender verify result: %+v", senderVerify)
	}

	senderTrust := runCLIJSON[trust.Report](t, binaryPath, senderDir, "trust-report", "--format", "json")
	if senderTrust.Status != trust.StatusPass || senderTrust.Gate.Status != trust.StatusPass {
		t.Fatalf("unexpected sender trust report: %+v", senderTrust)
	}

	encryptedRelativePath := filepath.Join("handoff", "acme-review.atb.enc")
	evidenceRelativePath := filepath.Join("handoff", "acme-review-evidence.zip")
	runCLIWithEnv(
		t,
		binaryPath,
		senderDir,
		passwordEnv,
		"encrypt",
		bundle.DefaultPath(),
		"--output",
		encryptedRelativePath,
	)
	runCLI(
		t,
		binaryPath,
		senderDir,
		"export",
		"--format",
		"compliance",
		"--output",
		evidenceRelativePath,
	)

	encryptedBytes, err := os.ReadFile(filepath.Join(senderDir, encryptedRelativePath))
	if err != nil {
		t.Fatalf("read encrypted handoff bundle: %v", err)
	}
	if bytes.Contains(encryptedBytes, []byte("claims-triage")) {
		t.Fatalf("expected encrypted handoff artefact to avoid plaintext event payloads")
	}

	zr, err := zip.OpenReader(filepath.Join(senderDir, evidenceRelativePath))
	if err != nil {
		t.Fatalf("open handoff evidence zip: %v", err)
	}
	defer zr.Close()

	requiredEntries := []string{
		"evidence/reports/trust-report.json",
		"evidence/reports/verify.json",
		"evidence/docs/spec-v1.0.md",
	}
	for _, entry := range requiredEntries {
		if !containsString(zipNames(zr.File), entry) {
			t.Fatalf("missing handoff evidence entry %q", entry)
		}
	}

	var exportedTrustReport trust.Report
	if err := json.Unmarshal(readZipFile(t, zr.File, "evidence/reports/trust-report.json"), &exportedTrustReport); err != nil {
		t.Fatalf("decode handoff trust report: %v", err)
	}
	if exportedTrustReport.Status != trust.StatusPass || exportedTrustReport.Gate.Status != trust.StatusPass {
		t.Fatalf("unexpected exported handoff trust report: %+v", exportedTrustReport)
	}

	recipientEncryptedPath := filepath.Join(recipientDir, "incoming", "acme-review.atb.enc")
	if err := copyTestFile(filepath.Join(senderDir, encryptedRelativePath), recipientEncryptedPath); err != nil {
		t.Fatalf("copy encrypted handoff artefact: %v", err)
	}

	recipientBundleRelativePath := filepath.Join("review", "acme-review.atb")
	runCLIWithEnv(
		t,
		binaryPath,
		recipientDir,
		passwordEnv,
		"decrypt",
		filepath.Join("incoming", "acme-review.atb.enc"),
		"--output",
		recipientBundleRelativePath,
	)

	recipientVerify := runCLIJSON[verifyResult](
		t,
		binaryPath,
		recipientDir,
		"verify",
		recipientBundleRelativePath,
		"--format",
		"json",
	)
	if recipientVerify.Status != "valid" || recipientVerify.ChainLength != 3 {
		t.Fatalf("unexpected recipient verify result: %+v", recipientVerify)
	}

	recipientTrust := runCLIJSON[trust.Report](
		t,
		binaryPath,
		recipientDir,
		"trust-report",
		recipientBundleRelativePath,
		"--format",
		"json",
	)
	if recipientTrust.Status != trust.StatusPass || recipientTrust.Gate.Status != trust.StatusPass {
		t.Fatalf("unexpected recipient trust report: %+v", recipientTrust)
	}

	receivedBundle, err := bundle.Load(filepath.Join(recipientDir, recipientBundleRelativePath))
	if err != nil {
		t.Fatalf("load recipient bundle: %v", err)
	}
	if len(receivedBundle.Records) != 3 {
		t.Fatalf("unexpected recipient bundle length: got %d want 3", len(receivedBundle.Records))
	}
	last := receivedBundle.Records[len(receivedBundle.Records)-1]
	if last.Event.Type != "snapshot.customer_handoff" {
		t.Fatalf("unexpected recipient final event type: %q", last.Event.Type)
	}
}

func copyTestFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil { // #nosec G301 -- test helper creates private temp dirs
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
