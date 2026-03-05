package trust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestBuildReportIncludesAllCategories(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "SECURITY.md"), "security")
	mustWriteFile(t, filepath.Join(root, "incident-response.md"), "incident")
	mustWriteFile(t, filepath.Join(root, "docs/security.md"), "docs")
	mustWriteFile(t, filepath.Join(root, "docs/spec-v1.0.md"), "spec")
	mustWriteFile(t, filepath.Join(root, "docs/quickstart.md"), "quickstart")
	mustWriteFile(t, filepath.Join(root, "cmd/atb/main_test.go"), "tests")
	mustWriteFile(t, filepath.Join(root, "test/golden/golden_test.go"), "oracle")

	bundlePath := filepath.Join(root, "run.atb", "bundle.atb")
	b := bundle.New()
	if err := b.Append("agent.session", map[string]interface{}{"id": "report-test"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save: %v", err)
	}

	report := BuildReport(root, bundlePath)
	if len(report.Categories) != 4 {
		t.Fatalf("expected 4 categories, got %d", len(report.Categories))
	}

	expected := map[string]bool{
		"cryptographic_integrity": false,
		"operational_safety":      false,
		"test_coverage":           false,
		"documentation":           false,
	}
	for _, category := range report.Categories {
		if _, ok := expected[category.Key]; ok {
			expected[category.Key] = true
		}
	}
	for key, seen := range expected {
		if !seen {
			t.Fatalf("missing category %q", key)
		}
	}

	if report.ChainLength != 1 {
		t.Fatalf("expected chain length 1, got %d", report.ChainLength)
	}
	if report.HeadHash == "" {
		t.Fatalf("expected non-empty head hash")
	}
	if report.Gate.Status != StatusPass {
		t.Fatalf("expected gate status pass, got %q", report.Gate.Status)
	}
	if report.Gate.BlockingFailures != 0 {
		t.Fatalf("expected zero blocking failures, got %d", report.Gate.BlockingFailures)
	}
	if report.Summary.Total == 0 {
		t.Fatalf("expected non-zero summary total")
	}
	if report.Summary.Total != report.Summary.Pass+report.Summary.Warn+report.Summary.Fail {
		t.Fatalf("summary counts do not add up: %+v", report.Summary)
	}
}

func TestBuildReportGateFailsOnTamperedChain(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "docs/spec-v1.0.md"), "spec")
	bundlePath := filepath.Join(root, "run.atb", "bundle.atb")

	b := bundle.New()
	if err := b.Append("agent.session", map[string]interface{}{"id": "tamper-test"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	b.Records[0].Hash = strings.Repeat("0", 64)
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save: %v", err)
	}

	report := BuildReport(root, bundlePath)
	if report.Gate.Status != StatusFail {
		t.Fatalf("expected gate status fail, got %q", report.Gate.Status)
	}
	if report.Gate.BlockingFailures < 1 {
		t.Fatalf("expected blocking failures, got %d", report.Gate.BlockingFailures)
	}
	found := false
	for _, id := range report.Gate.FailedChecks {
		if id == "cryptographic_integrity.hash_chain" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected failed blocking check cryptographic_integrity.hash_chain, got %v", report.Gate.FailedChecks)
	}
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
