package trust

import (
	"os"
	"path/filepath"
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
