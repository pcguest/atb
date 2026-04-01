// Package trust builds high-level trust reports for ATB bundles and shipped evidence.
package trust

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	atbembed "github.com/pcguest/atb"
	"github.com/pcguest/atb/internal/bundle"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusWarn = "warn"

	SeverityCritical = "critical"
	SeverityAdvisory = "advisory"
)

// Check captures one auditable trust check with optional evidence paths.
type Check struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Severity string   `json:"severity"`
	Blocking bool     `json:"blocking"`
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

// Gate captures whether all blocking checks passed.
type Gate struct {
	Status           string   `json:"status"`
	BlockingFailures int      `json:"blocking_failures"`
	FailedChecks     []string `json:"failed_checks,omitempty"`
}

// Summary aggregates report check counts.
type Summary struct {
	Total int `json:"total"`
	Pass  int `json:"pass"`
	Warn  int `json:"warn"`
	Fail  int `json:"fail"`
}

// Report is the machine-readable trust report envelope.
type Report struct {
	Status      string      `json:"status"`
	GeneratedAt string      `json:"generated_at"`
	BundlePath  string      `json:"bundle_path"`
	ChainLength int         `json:"chain_length"`
	HeadHash    string      `json:"head_hash,omitempty"`
	Gate        Gate        `json:"gate"`
	Summary     Summary     `json:"summary"`
	Categories  []Category  `json:"categories"`
	CAS         *CASSection `json:"cas,omitempty"`
}

// BuildReport creates a trust report for the bundle path, preferring workspace
// evidence under repoRoot and falling back to shipped embedded evidence.
func BuildReport(repoRoot string, bundlePath string, profileID string) Report {
	categories := []Category{
		cryptoCategory(bundlePath, repoRoot),
		operationalSafetyCategory(repoRoot),
		testCoverageCategory(repoRoot),
		documentationCategory(repoRoot),
	}
	if profileCategory, ok := profileVerificationCategory(bundlePath, profileID); ok {
		categories = append(categories, profileCategory)
	}
	casSection, casWarning := buildCASSection(bundlePath, profileID)
	if casWarning != "" {
		categories = append(categories, casProfileWarningCategory(casWarning))
	}
	report := Report{
		Status:      aggregateStatus(categories),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		BundlePath:  bundlePath,
		Gate:        evaluateGate(categories),
		Summary:     summarizeChecks(categories),
		Categories:  categories,
		CAS:         casSection,
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

func profileVerificationCategory(bundlePath string, profileID string) (Category, bool) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return Category{}, false
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		return Category{}, false
	}

	report := verifypkg.Verify(b, bundlePath, profileID)
	if len(report.Profiles) == 0 {
		return Category{}, false
	}

	profile := report.Profiles[0]
	checks := make([]Check, 0, len(profile.CriticalFailures)+len(profile.RequiredWarnings)+1)
	checks = append(checks, Check{
		ID:       "profile_verification",
		Title:    "Profile Verification",
		Status:   StatusPass,
		Severity: SeverityCritical,
		Blocking: true,
		Details:  fmt.Sprintf("Profile %s evaluated.", profile.ProfileID),
	})

	for i, failure := range profile.CriticalFailures {
		checks = append(checks, Check{
			ID:       fmt.Sprintf("profile_failure_%d", i+1),
			Title:    "Profile Critical Failure",
			Status:   StatusFail,
			Severity: SeverityCritical,
			Blocking: true,
			Details:  fmt.Sprintf("%s: %s", failure.Kind, failure.Detail),
		})
	}
	for i, warning := range profile.RequiredWarnings {
		checks = append(checks, Check{
			ID:       fmt.Sprintf("profile_warning_%d", i+1),
			Title:    "Profile Required Warning",
			Status:   StatusWarn,
			Severity: SeverityAdvisory,
			Blocking: false,
			Details:  warning,
		})
	}

	return Category{
		Key:    "obligation_profile",
		Title:  "Obligation Profile",
		Status: aggregateChecksStatus(checks),
		Checks: checks,
	}, true
}

func cryptoCategory(bundlePath string, repoRoot string) Category {
	checks := []Check{
		evidenceCheck(
			"canonicalization_profile",
			"Canonicalization Profile",
			repoRoot,
			"docs/spec-v1.0.md",
			"ATB includes the RFC 8785-compatible canonicalization specification.",
			"RFC8785-compatible canonicalization spec is missing (expected docs/spec-v1.0.md).",
			SeverityAdvisory,
			false,
		),
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		checks = append(checks, Check{
			ID:       "hash_chain",
			Title:    "Hash Chain Verification",
			Status:   StatusFail,
			Severity: SeverityCritical,
			Blocking: true,
			Details:  fmt.Sprintf("Unable to load bundle for verification: %v", err),
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
			ID:       "hash_chain",
			Title:    "Hash Chain Verification",
			Status:   StatusFail,
			Severity: SeverityCritical,
			Blocking: true,
			Details:  "Bundle is empty; no events available for chain verification.",
			Evidence: []string{
				bundlePath,
			},
		})
	} else if err := b.Verify(); err != nil {
		checks = append(checks, Check{
			ID:       "hash_chain",
			Title:    "Hash Chain Verification",
			Status:   StatusFail,
			Severity: SeverityCritical,
			Blocking: true,
			Details:  fmt.Sprintf("Hash chain verification failed: %v", err),
			Evidence: []string{
				bundlePath,
			},
		})
	} else {
		checks = append(checks, Check{
			ID:       "hash_chain",
			Title:    "Hash Chain Verification",
			Status:   StatusPass,
			Severity: SeverityCritical,
			Blocking: true,
			Details:  fmt.Sprintf("Verified %d event(s); chain is intact.", len(b.Records)),
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
		evidenceCheck(
			"security_policy",
			"Security Policy",
			repoRoot,
			"SECURITY.md",
			"ATB includes a public security policy.",
			"Security policy is missing (expected SECURITY.md).",
			SeverityAdvisory,
			false,
		),
		evidenceCheck(
			"incident_response",
			"Incident Response",
			repoRoot,
			"incident-response.md",
			"ATB includes an incident response runbook.",
			"Incident response runbook is missing (expected incident-response.md).",
			SeverityAdvisory,
			false,
		),
		evidenceCheck(
			"security_docs",
			"Security Hardening Docs",
			repoRoot,
			"docs/security.md",
			"ATB includes operational security guidance.",
			"Operational security guidance is missing (expected docs/security.md).",
			SeverityAdvisory,
			false,
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
		evidenceCheck(
			"go_tests",
			"Go Test Suite",
			repoRoot,
			"cmd/atb/main_test.go",
			"ATB includes the core CLI test suite definition.",
			"Core CLI test file is missing (expected cmd/atb/main_test.go).",
			SeverityAdvisory,
			false,
		),
		evidenceCheck(
			"cross_language_oracle",
			"Cross-Language Oracle",
			repoRoot,
			"test/golden/golden_test.go",
			"ATB includes the cross-language canonicalization oracle test.",
			"Cross-language oracle test is missing (expected test/golden/golden_test.go).",
			SeverityAdvisory,
			false,
		),
		evidenceCheck(
			"python_property_tests",
			"Python Property Tests",
			repoRoot,
			"sdk/python/tests/test_properties.py",
			"ATB includes the Python property test template for SDK invariants.",
			"Property-based test template is missing (expected sdk/python/tests/test_properties.py).",
			SeverityAdvisory,
			false,
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
		evidenceCheck(
			"quickstart",
			"Quickstart Guide",
			repoRoot,
			"docs/quickstart.md",
			"ATB includes end-user quickstart documentation.",
			"Quickstart guide is missing (expected docs/quickstart.md).",
			SeverityAdvisory,
			false,
		),
		evidenceCheck(
			"ai_integration",
			"AI Integration Guide",
			repoRoot,
			"docs/ai-integration.md",
			"ATB includes the AI integration contract documentation.",
			"AI integration contract doc is missing (expected docs/ai-integration.md).",
			SeverityAdvisory,
			false,
		),
		evidenceCheck(
			"event_schema",
			"Event Schema",
			repoRoot,
			"schemas/event.v1.json",
			"ATB includes the event JSON schema.",
			"Event schema is missing (expected schemas/event.v1.json).",
			SeverityAdvisory,
			false,
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

func evidenceCheck(
	id string,
	title string,
	repoRoot string,
	repoRelative string,
	passDetails string,
	warnDetails string,
	severity string,
	blocking bool,
) Check {
	check := Check{
		ID:       id,
		Title:    title,
		Status:   StatusWarn,
		Severity: severity,
		Blocking: blocking,
		Details:  warnDetails,
	}
	if evidencePath, ok := findEvidencePath(repoRoot, repoRelative); ok {
		check.Status = StatusPass
		check.Details = passDetails
		check.Evidence = []string{evidencePath}
	}
	return check
}

func findEvidencePath(repoRoot string, repoRelative string) (string, bool) {
	trimmedRoot := strings.TrimSpace(repoRoot)
	if trimmedRoot != "" {
		path := filepath.Join(trimmedRoot, filepath.FromSlash(repoRelative))
		if fileExists(path) {
			return path, true
		}
	}

	if atbembed.HasEmbeddedTrustEvidence(repoRelative) {
		return repoRelative, true
	}

	return "", false
}

func evaluateGate(categories []Category) Gate {
	gate := Gate{Status: StatusPass}
	for _, category := range categories {
		for _, check := range category.Checks {
			if check.Blocking && check.Status != StatusPass {
				gate.Status = StatusFail
				gate.BlockingFailures++
				gate.FailedChecks = append(gate.FailedChecks, fmt.Sprintf("%s.%s", category.Key, check.ID))
			}
		}
	}
	return gate
}

func summarizeChecks(categories []Category) Summary {
	summary := Summary{}
	for _, category := range categories {
		for _, check := range category.Checks {
			summary.Total++
			switch check.Status {
			case StatusPass:
				summary.Pass++
			case StatusWarn:
				summary.Warn++
			case StatusFail:
				summary.Fail++
			}
		}
	}
	return summary
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
