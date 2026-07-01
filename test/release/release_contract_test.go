package release_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func readRepositoryFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func makeTarget(t *testing.T, makefile, target, nextTarget string) string {
	t.Helper()
	startMarker := "\n" + target + ":"
	start := strings.Index(makefile, startMarker)
	if start < 0 {
		t.Fatalf("Makefile target %s not found", target)
	}
	end := strings.Index(makefile[start+len(startMarker):], "\n"+nextTarget+":")
	if end < 0 {
		t.Fatalf("Makefile target %s after %s not found", nextTarget, target)
	}
	return makefile[start : start+len(startMarker)+end]
}

func TestEventV1SchemaMatchesFrozenDigest(t *testing.T) {
	root := repositoryRoot(t)
	schema, err := os.ReadFile(filepath.Join(root, "schemas", "event.v1.json"))
	if err != nil {
		t.Fatalf("read frozen event schema: %v", err)
	}
	manifest := strings.TrimSpace(readRepositoryFile(t, "schemas/event.v1.sha256"))
	got := fmt.Sprintf("%x  event.v1.json", sha256.Sum256(schema))
	if manifest != got {
		t.Fatalf("event.v1 schema digest mismatch\nmanifest: %s\nactual:   %s", manifest, got)
	}
}

func TestGoldReleaseGateHasNoSoftFailurePaths(t *testing.T) {
	makefile := "\n" + readRepositoryFile(t, "Makefile")
	goldGate := makeTarget(t, makefile, "gate-gold-release", "deps-update")

	for _, forbidden := range []string{
		"@-make security-scan",
		"Security scan warning (non-blocking)",
		"CYPRESS_MOCK_API",
		"Lighthouse skipped",
	} {
		if strings.Contains(goldGate, forbidden) {
			t.Errorf("gold release gate contains soft-failure path %q", forbidden)
		}
	}
	for _, required := range []string{
		"$(MAKE) security-scan",
		"$(MAKE) test-e2e",
		"npm run test:a11y",
	} {
		if !strings.Contains(goldGate, required) {
			t.Errorf("gold release gate does not require %q", required)
		}
	}

	e2e := makeTarget(t, makefile, "test-e2e", "test-performance")
	if strings.Contains(e2e, "npm install") {
		t.Error("test-e2e uses npm install instead of reproducible npm ci")
	}
	if !strings.Contains(e2e, "npm ci") {
		t.Error("test-e2e does not install dependencies with npm ci")
	}

	coverage := makeTarget(t, makefile, "coverage-check", "test-embed")
	if !strings.Contains(coverage, "set -eu;") {
		t.Error("coverage-check does not fail fast when the threshold command fails")
	}
}

func TestReleaseWorkflowPublishesCompleteCanonicalAssets(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/release.yml")
	required := []string{
		"artifacts/cli/atb-linux-amd64",
		"artifacts/cli/atb-windows-amd64.exe",
		"artifacts/python/*",
		"artifacts/npm/*",
		"artifacts/web/*",
		"artifacts/sbom/*",
		"artifacts/provenance/*",
		"draft: true",
		"gh release edit \"${GITHUB_REF_NAME}\" --draft=false",
	}
	for _, value := range required {
		if !strings.Contains(workflow, value) {
			t.Errorf("release workflow does not publish required canonical asset or transition %q", value)
		}
	}
}

func TestReleaseSecurityToolsAreVersionPinned(t *testing.T) {
	for _, name := range []string{
		"Makefile",
		".github/workflows/ci.yml",
		".github/workflows/security.yml",
	} {
		if strings.Contains(readRepositoryFile(t, name), "@latest") {
			t.Errorf("%s installs an unpinned security tool with @latest", name)
		}
	}
}

func TestSecurityScanExcludesRepositoryBuildCaches(t *testing.T) {
	makefile := readRepositoryFile(t, "Makefile")
	start := strings.Index(makefile, "\nsecurity-scan:")
	if start < 0 {
		t.Fatal("Makefile security-scan target not found")
	}
	securityTarget := makefile[start:]
	for _, dir := range []string{".gocache", ".tmp"} {
		flag := "--skip-dirs " + dir
		if count := strings.Count(securityTarget, flag); count != 2 {
			t.Errorf("security-scan must apply %q to native and Docker Trivy commands; found %d", flag, count)
		}
	}
	if strings.Contains(securityTarget, "gosec ./...") {
		t.Error("security-scan recursively scans build caches with gosec ./...")
	}
	if !strings.Contains(makefile, "GOSEC_DIRS =") ||
		!strings.Contains(makefile, `go list -f '{{.Dir}}' ./...`) {
		t.Error("Makefile does not derive filesystem directories for gosec")
	}
	if !strings.Contains(makefile, `grep -v '/node_modules/'`) {
		t.Error("Makefile does not exclude dependency directories from gosec")
	}
	if !strings.Contains(securityTarget, `"$$GOSEC_BIN" $(GOSEC_DIRS)`) {
		t.Error("native gosec scan does not use the module package directories")
	}
	if !strings.Contains(securityTarget, `/go/bin/gosec $$(go list -f "{{.Dir}}" ./...`) {
		t.Error("Docker gosec scan does not derive the module package directories")
	}
}

func TestGoldGateChecksPreparedVersionMarkers(t *testing.T) {
	makefile := readRepositoryFile(t, "Makefile")
	if !strings.Contains(makefile, "check-versions:\n") {
		t.Fatal("Makefile has no local version-consistency target")
	}
	if !strings.Contains(makefile, "hygiene-quick: check-generated check-versions") {
		t.Fatal("hygiene-quick does not enforce version consistency")
	}

	script := readRepositoryFile(t, "scripts/check-versions.sh")
	for _, marker := range []string{
		"sdk/typescript/src/index.ts",
		"README.md",
		"SECURITY.md",
		"CHANGELOG.md",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("version check does not cover %s", marker)
		}
	}
}
