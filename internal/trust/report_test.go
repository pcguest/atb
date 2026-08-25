// SPDX-License-Identifier: MIT
package trust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestBuildReportIncludesAllCategories(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "SECURITY.md"), "security")
	mustWriteFile(t, filepath.Join(root, "SECURITY.md"), "incident")
	mustWriteFile(t, filepath.Join(root, "docs/concepts/trust-model.md"), "docs")
	mustWriteFile(t, filepath.Join(root, "docs/specification/bundle-v1.md"), "spec")
	mustWriteFile(t, filepath.Join(root, "docs/getting-started/quickstart.md"), "quickstart")
	mustWriteFile(t, filepath.Join(root, "cmd/atb/main_test.go"), "tests")
	mustWriteFile(t, filepath.Join(root, "test/golden/golden_test.go"), "oracle")

	bundlePath := filepath.Join(root, "bundle.atb")
	b := newTrustTestBundle(t)
	timestamp := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if err := b.AppendWithOptions("agent.session", map[string]interface{}{"id": "report-test"}, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save: %v", err)
	}

	report := BuildReport(root, bundlePath, "")
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

	if report.ChainLength != 2 {
		t.Fatalf("expected chain length 2, got %d", report.ChainLength)
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

	mustWriteFile(t, filepath.Join(root, "docs/specification/bundle-v1.md"), "spec")
	bundlePath := filepath.Join(root, "bundle.atb")

	b := newTrustTestBundle(t)
	timestamp := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if err := b.AppendWithOptions("agent.session", map[string]interface{}{"id": "tamper-test"}, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append: %v", err)
	}
	b.Records[0].Hash = strings.Repeat("0", 64)
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save: %v", err)
	}

	report := BuildReport(root, bundlePath, "")
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

func TestBuildReportPortableModeUsesEmbeddedEvidence(t *testing.T) {
	root := t.TempDir()
	bundlePath := filepath.Join(root, "bundle.atb")

	b := newTrustTestBundle(t)
	timestamp := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if err := b.AppendWithOptions("agent.session", map[string]interface{}{"id": "portable-report"}, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save: %v", err)
	}

	report := BuildReport(root, bundlePath, "")
	if report.Status != StatusPass {
		t.Fatalf("expected portable report status pass, got %q", report.Status)
	}
	if report.Gate.Status != StatusPass {
		t.Fatalf("expected portable gate pass, got %q", report.Gate.Status)
	}
	if report.Summary.Warn != 0 || report.Summary.Fail != 0 {
		t.Fatalf("expected portable report without warnings or failures, got %+v", report.Summary)
	}

	required := map[string]string{
		"canonicalization_profile": "docs/specification/bundle-v1.md",
		"security_policy":          "SECURITY.md",
		"incident_response":        "SECURITY.md",
		"security_docs":            "docs/concepts/trust-model.md",
		"go_tests":                 "cmd/atb/main_test.go",
		"cross_language_oracle":    "test/golden/golden_test.go",
		"python_property_tests":    "sdk/python/tests/test_properties.py",
		"quickstart":               "docs/getting-started/quickstart.md",
		"ai_integration":           "docs/maintainers/automation-contract.md",
		"event_schema":             "schemas/event.v1.json",
	}
	for _, category := range report.Categories {
		for _, check := range category.Checks {
			expectedEvidence, ok := required[check.ID]
			if !ok || check.ID == "hash_chain" {
				continue
			}
			if check.Status != StatusPass {
				t.Fatalf("expected %s to pass in portable mode, got %q", check.ID, check.Status)
			}
			if len(check.Evidence) != 1 || check.Evidence[0] != expectedEvidence {
				t.Fatalf("expected %s evidence %q, got %v", check.ID, expectedEvidence, check.Evidence)
			}
		}
	}
}

func TestBuildReport_WithProfileEmitsCAS(t *testing.T) {
	bundlePath := writePrivilegedToolActionBundle(t)

	report := BuildReport("", bundlePath, "atb.profile.privileged_tool_action")
	if report.CAS == nil {
		t.Fatalf("expected CAS section")
	}
	if report.CAS.ProfileID != "atb.profile.privileged_tool_action" {
		t.Fatalf("unexpected CAS profile ID: got %q want %q", report.CAS.ProfileID, "atb.profile.privileged_tool_action")
	}
	if report.CAS.AnchorQuality.Label != "absent" {
		t.Fatalf("unexpected anchor quality label: got %q want %q", report.CAS.AnchorQuality.Label, "absent")
	}
}

func TestBuildReport_WithProfileEmitsCASForDataExport(t *testing.T) {
	bundlePath := writeDataExportBundle(t)

	report := BuildReport("", bundlePath, "atb.profile.data_export")
	if report.CAS == nil {
		t.Fatalf("expected CAS section, got nil")
	}
}

func TestBuildReport_WithProfileIncludesCriticalFailures(t *testing.T) {
	bundlePath := writePrivilegedToolActionBundle(t)

	report := BuildReport("", bundlePath, "atb.profile.privileged_tool_action")
	if report.Status != StatusFail {
		t.Fatalf("expected failing report status, got %q", report.Status)
	}
	if report.Gate.Status != StatusFail {
		t.Fatalf("expected failing gate status, got %q", report.Gate.Status)
	}
	if !hasCheckDetail(report, "obligation_profile", "missing_event: ai.human.approval required when actions execute") {
		t.Fatalf("expected missing approval detail in obligation profile category, got %+v", report.Categories)
	}
}

func TestBuildReport_WithoutProfileNoCAS(t *testing.T) {
	bundlePath := writePrivilegedToolActionBundle(t)

	report := BuildReport("", bundlePath, "")
	if report.CAS != nil {
		t.Fatalf("expected CAS section to be omitted, got %+v", report.CAS)
	}
}

func writePrivilegedToolActionBundle(t testing.TB) string {
	t.Helper()

	bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
	b := newTrustTestBundle(t)

	appendTrustRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "lookup-status",
	}, "2026-03-27T12:00:00Z")
	appendTrustRecord(t, b, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-1",
		"action_type":              "lookup_status",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "fetch status",
	}, "2026-03-27T12:01:00Z")
	appendTrustRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}, "2026-03-27T12:02:00Z")
	appendTrustRecord(t, b, event.TypeAIActionExecuted, map[string]any{
		"action_id":           "act-1",
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-03-27T12:03:00Z")
	appendTrustRecord(t, b, event.TypeAIActionCommitted, map[string]any{
		"action_id":           "act-1",
		"commit_outcome":      "success",
		"sink_receipt_digest": "sink-digest",
	}, "2026-03-27T12:04:00Z")

	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return bundlePath
}

func writeDataExportBundle(t testing.TB) string {
	t.Helper()

	bundlePath := filepath.Join(t.TempDir(), "bundle.atb")
	b := newTrustTestBundle(t)

	appendTrustRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "data_export",
	}, "2026-03-27T12:00:00Z")
	appendTrustRecord(t, b, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "export-1",
		"action_type":              "export_data",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "customer-dataset",
		"intended_effect":          "export dataset",
	}, "2026-03-27T12:01:00Z")
	appendTrustRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "export-1",
	}, "2026-03-27T12:02:00Z")
	appendTrustRecord(t, b, event.TypeDataExportPrecommit, map[string]any{
		"export_id":              "export-1",
		"destination":            "s3://exports/customer",
		"format":                 "jsonl",
		"approval_ticket":        "TICK-1",
		"subject_scope_declared": true,
		"subject_id_hash":        "subject-hash",
	}, "2026-03-27T12:03:00Z")
	appendTrustRecord(t, b, event.TypeAIHumanApproval, map[string]any{
		"approval_id":          "approval-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approved",
		"justification_digest": "justification-digest",
		"action_id":            "export-1",
	}, "2026-03-27T12:04:00Z")
	appendTrustRecord(t, b, event.TypeDataExportExecuted, map[string]any{
		"export_id":                "export-1",
		"destination_receipt_hash": "receipt-hash",
		"row_count":                10,
	}, "2026-03-27T12:05:00Z")

	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return bundlePath
}

func appendTrustRecord(t testing.TB, b *bundle.Bundle, eventType string, data any, timestamp string) {
	t.Helper()
	if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
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

func hasCheckDetail(report Report, categoryKey string, want string) bool {
	for _, category := range report.Categories {
		if category.Key != categoryKey {
			continue
		}
		for _, check := range category.Checks {
			if check.Details == want {
				return true
			}
		}
	}
	return false
}
