// SPDX-License-Identifier: MIT
package custody_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/pkg/custody"
)

func TestNewBundleExportSerialisesVerifiedBundle(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "bundles", "profiles", "rag_answer-pass.atb")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	submittedAt := time.Date(2026, 5, 28, 1, 2, 3, 0, time.UTC)
	export, err := custody.NewBundleExport(path, custody.ExportOptions{
		ProfileID:    "atb.profile.rag_answer",
		SubmitterRef: "local-submit-1",
		SubmittedAt:  submittedAt,
	})
	if err != nil {
		t.Fatalf("NewBundleExport: %v", err)
	}
	if export.ExportVersion != custody.BundleExportVersion {
		t.Fatalf("ExportVersion = %q, want %q", export.ExportVersion, custody.BundleExportVersion)
	}
	if export.BundleHash == "" {
		t.Fatal("expected bundle hash")
	}
	if export.ReceiptID != "sha256-"+export.BundleHash {
		t.Fatalf("ReceiptID = %q, want sha256-%s", export.ReceiptID, export.BundleHash)
	}
	if export.SubmittedAt != "2026-05-28T01:02:03Z" {
		t.Fatalf("SubmittedAt = %q", export.SubmittedAt)
	}
	if !bytes.Equal(export.Bundle, raw) {
		t.Fatal("bundle bytes changed during export")
	}
	if export.VerifyReport.ReportVersion != custody.VerifyReportVersion {
		t.Fatalf("report_version = %q, want %q", export.VerifyReport.ReportVersion, custody.VerifyReportVersion)
	}
	if !export.VerifyReport.Pass {
		t.Fatalf("expected passing verifier report, failures=%+v", export.VerifyReport.Failures)
	}

	wire, err := export.Export()
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	var roundTrip custody.BundleExport
	if err := json.Unmarshal(wire, &roundTrip); err != nil {
		t.Fatalf("unmarshal wire export: %v", err)
	}
	if roundTrip.ReceiptID != export.ReceiptID {
		t.Fatalf("round-trip receipt_id = %q, want %q", roundTrip.ReceiptID, export.ReceiptID)
	}
	if !bytes.Equal(roundTrip.Bundle, raw) {
		t.Fatal("round-trip bundle bytes changed")
	}
}

func TestNewBundleExportRejectsTamperedBundle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	if err := b.Append("dev.session", map[string]any{"step": "before"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	tampered := bytes.Replace(raw, []byte("dev.session"), []byte("dev.tamperx"), 1)
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatalf("write tampered bundle: %v", err)
	}

	_, err = custody.NewBundleExport(path, custody.ExportOptions{})
	if err == nil {
		t.Fatal("expected tampered bundle export to fail")
	}
}

func TestBundleExportRejectsIncompleteWireObject(t *testing.T) {
	_, err := (custody.BundleExport{}).Export()
	if !errors.Is(err, custody.ErrInvalidExport) {
		t.Fatalf("Export() error = %v, want %v", err, custody.ErrInvalidExport)
	}
}
