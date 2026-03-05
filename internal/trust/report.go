// Package trust builds high-level trust reports for ATB repositories.
package trust

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusWarn = "warn"
)

// Check captures one auditable trust check with optional evidence paths.
type Check struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Details  string   `json:"details"`
	Evidence []string `json:"evidence,omitempty"`
}

// Category groups related trust checks.
type Category struct {
	Key    string  `json:"key"`
	Title  string  `json:"title"`
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}

// Report is the machine-readable trust report envelope.
type Report struct {
	Status      string     `json:"status"`
	GeneratedAt string     `json:"generated_at"`
	BundlePath  string     `json:"bundle_path"`
	ChainLength int        `json:"chain_length"`
	HeadHash    string     `json:"head_hash,omitempty"`
	Categories  []Category `json:"categories"`
}

// BuildReport creates a trust report for the bundle path rooted at repoRoot.
func BuildReport(repoRoot string, bundlePath string) Report {
	categories := []Category{
		cryptoCategory(bundlePath, repoRoot),
		operationalSafetyCategory(repoRoot),
		testCoverageCategory(repoRoot),
		documentationCategory(repoRoot),
	}
	report := Report{
		Status:      aggregateStatus(categories),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		BundlePath:  bundlePath,
		Categories:  categories,
	}
	for _, category := range categories {
		if category.Key != "cryptographic_integrity" {
			continue
		}
		for _, check := range category.Checks {
			if check.ID == "hash_chain" && check.Status == StatusPass {
				b, err := bundle.Load(bundlePath)
				if err == nil {
					report.ChainLength = len(b.Records)
					if len(b.Records) > 0 {
						report.HeadHash = b.Records[len(b.Records)-1].Hash
					}
				}
			}
		}
	}
	return report
}

func cryptoCategory(bundlePath string, repoRoot string) Category {
	checks := []Check{
		presenceCheck(
			"canonicalization_profile",
			"Canonicalization Profile",
			filepath.Join(repoRoot, "docs/spec-v1.0.md"),
			"RFC8785-compatible canonicalization spec is present.",
			"RFC8785-compatible canonicalization spec is missing (expected docs/spec-v1.0.md).",
		),
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		checks = append(checks, Check{
			ID:      "hash_chain",
			Title:   "Hash Chain Verification",
			Status:  StatusFail,
			Details: fmt.Sprintf("Unable to load bundle for verification: %v", err),
			Evidence: []string{
				bundlePath,
			},
		})
		return Category{
			Key:    "cryptographic_integrity",
			Title:  "Cryptographic Integrity",
			Status: aggregateChecksStatus(checks),
			Checks: checks,
		}
	}
	if len(b.Records) == 0 {
		checks = append(checks, Check{
			ID:      "hash_chain",
			Title:   "Hash Chain Verification",
			Status:  StatusWarn,
			Details: "Bundle is empty; no events available for chain verification.",
			Evidence: []string{
				bundlePath,
			},
		})
	} else if err := b.Verify(); err != nil {
		checks = append(checks, Check{
			ID:      "hash_chain",
			Title:   "Hash Chain Verification",
			Status:  StatusFail,
			Details: fmt.Sprintf("Hash chain verification failed: %v", err),
			Evidence: []string{
				bundlePath,
			},
		})
	} else {
		checks = append(checks, Check{
			ID:      "hash_chain",
			Title:   "Hash Chain Verification",
			Status:  StatusPass,
			Details: fmt.Sprintf("Verified %d event(s); chain is intact.", len(b.Records)),
			Evidence: []string{
				bundlePath,
			},
		})
	}

	return Category{
		Key:    "cryptographic_integrity",
		Title:  "Cryptographic Integrity",
		Status: aggregateChecksStatus(checks),
		Checks: checks,
	}
}

func operationalSafetyCategory(repoRoot string) Category {
	checks := []Check{
		presenceCheck(
			"security_policy",
			"Security Policy",
			filepath.Join(repoRoot, "SECURITY.md"),
			"Repository includes a public security policy.",
			"Security policy is missing (expected SECURITY.md).",
		),
		presenceCheck(
			"incident_response",
			"Incident Response",
			filepath.Join(repoRoot, "incident-response.md"),
			"Repository includes an incident response runbook.",
			"Incident response runbook is missing (expected incident-response.md).",
		),
		presenceCheck(
			"security_docs",
			"Security Hardening Docs",
			filepath.Join(repoRoot, "docs/security.md"),
			"Operational security guidance is documented.",
			"Operational security guidance is missing (expected docs/security.md).",
		),
	}
	return Category{
		Key:    "operational_safety",
		Title:  "Operational Safety",
		Status: aggregateChecksStatus(checks),
		Checks: checks,
	}
}

func testCoverageCategory(repoRoot string) Category {
	checks := []Check{
		presenceCheck(
			"go_tests",
			"Go Test Suite",
			filepath.Join(repoRoot, "cmd/atb/main_test.go"),
			"Core CLI package includes Go tests.",
			"Core CLI test file is missing (expected cmd/atb/main_test.go).",
		),
		presenceCheck(
			"cross_language_oracle",
			"Cross-Language Oracle",
			filepath.Join(repoRoot, "test/golden/golden_test.go"),
			"Cross-language canonicalization oracle test exists.",
			"Cross-language oracle test is missing (expected test/golden/golden_test.go).",
		),
		presenceCheck(
			"python_property_tests",
			"Python Property Tests",
			filepath.Join(repoRoot, "sdk/python/tests/test_properties.py"),
			"Property-based test template is available for AI extension.",
			"Property-based test template is missing (expected sdk/python/tests/test_properties.py).",
		),
	}
	return Category{
		Key:    "test_coverage",
		Title:  "Test Coverage",
		Status: aggregateChecksStatus(checks),
		Checks: checks,
	}
}

func documentationCategory(repoRoot string) Category {
	checks := []Check{
		presenceCheck(
			"quickstart",
			"Quickstart Guide",
			filepath.Join(repoRoot, "docs/quickstart.md"),
			"End-user quickstart documentation is present.",
			"Quickstart guide is missing (expected docs/quickstart.md).",
		),
		presenceCheck(
			"ai_integration",
			"AI Integration Guide",
			filepath.Join(repoRoot, "docs/ai-integration.md"),
			"AI integration contract documentation is present.",
			"AI integration contract doc is missing (expected docs/ai-integration.md).",
		),
		presenceCheck(
			"event_schema",
			"Event Schema",
			filepath.Join(repoRoot, "schemas/event.v1.json"),
			"Event JSON schema is available.",
			"Event schema is missing (expected schemas/event.v1.json).",
		),
	}
	return Category{
		Key:    "documentation",
		Title:  "Documentation",
		Status: aggregateChecksStatus(checks),
		Checks: checks,
	}
}

func aggregateStatus(categories []Category) string {
	hasWarn := false
	for _, category := range categories {
		switch category.Status {
		case StatusFail:
			return StatusFail
		case StatusWarn:
			hasWarn = true
		}
	}
	if hasWarn {
		return StatusWarn
	}
	return StatusPass
}

func aggregateChecksStatus(checks []Check) string {
	hasWarn := false
	for _, check := range checks {
		switch check.Status {
		case StatusFail:
			return StatusFail
		case StatusWarn:
			hasWarn = true
		}
	}
	if hasWarn {
		return StatusWarn
	}
	return StatusPass
}

func presenceCheck(id string, title string, path string, passDetails string, warnDetails string) Check {
	check := Check{
		ID:      id,
		Title:   title,
		Status:  StatusWarn,
		Details: warnDetails,
	}
	if fileExists(path) {
		check.Status = StatusPass
		check.Details = passDetails
		check.Evidence = []string{path}
	}
	return check
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
