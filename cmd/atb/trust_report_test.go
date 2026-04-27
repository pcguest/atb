// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/trust"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestTrustReportParseArgs(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "custom.atb")

	tests := []struct {
		name    string
		args    []string
		want    trustReportConfig
		wantErr bool
	}{
		{
			name: "defaults",
			want: trustReportConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     "markdown",
				ProfileID:  "",
			},
		},
		{
			name: "json format",
			args: []string{"--format", "json"},
			want: trustReportConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     "json",
				ProfileID:  "",
			},
		},
		{
			name: "text format",
			args: []string{"--format", "text"},
			want: trustReportConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     "text",
				ProfileID:  "",
			},
		},
		{
			name: "path and equals-format",
			args: []string{custom, "--format=markdown"},
			want: trustReportConfig{
				BundlePath: custom,
				Format:     "markdown",
				ProfileID:  "",
			},
		},
		{
			name: "profile flag",
			args: []string{"--profile", "atb.profile.privileged_tool_action"},
			want: trustReportConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     "markdown",
				ProfileID:  "atb.profile.privileged_tool_action",
			},
		},
		{
			name: "equals profile flag",
			args: []string{"--profile=atb.profile.rag_answer"},
			want: trustReportConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     "markdown",
				ProfileID:  "atb.profile.rag_answer",
			},
		},
		{
			name:    "missing format value",
			args:    []string{"--format"},
			wantErr: true,
		},
		{
			name:    "missing profile value",
			args:    []string{"--profile"},
			wantErr: true,
		},
		{
			name:    "unknown format",
			args:    []string{"--format", "yaml"},
			wantErr: true,
		},
		{
			name:    "too many paths",
			args:    []string{"one.atb", "two.atb"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--wat"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTrustReportArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected config: got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestTrustReportRenderTextWithoutCAS(t *testing.T) {
	report := trust.Report{
		Status:     trust.StatusPass,
		BundlePath: "/tmp/run.atb/bundle.atb",
		Gate: trust.Gate{
			Status: trust.StatusPass,
		},
		Summary: trust.Summary{
			Total: 7,
			Pass:  6,
			Warn:  1,
			Fail:  0,
		},
		Categories: []trust.Category{
			{Key: "cryptographic_integrity", Status: trust.StatusPass},
			{Key: "documentation", Status: trust.StatusWarn},
		},
	}

	var out bytes.Buffer
	renderTrustReportText(&out, report)
	got := out.String()

	for _, want := range []string{
		"Bundle:   /tmp/run.atb/bundle.atb",
		"Status:   PASS",
		"Gate:     PASS",
		"Summary:  total=7 pass=6 warn=1 fail=0",
		"Categories:",
		"  cryptographic_integrity: PASS",
		"  documentation: WARN",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderTrustReportText() missing %q in output:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\nCAS:\n") {
		t.Fatalf("renderTrustReportText() unexpectedly rendered CAS block:\n%s", got)
	}
}

func TestTrustReportRenderTextWithCAS(t *testing.T) {
	report := trust.Report{
		Status:     trust.StatusWarn,
		BundlePath: "/tmp/run.atb/bundle.atb",
		Gate: trust.Gate{
			Status: trust.StatusWarn,
		},
		Summary: trust.Summary{
			Total: 8,
			Pass:  5,
			Warn:  2,
			Fail:  1,
		},
		Categories: []trust.Category{
			{Key: "cryptographic_integrity", Status: trust.StatusPass},
		},
		CAS: &trust.CASSection{
			ProfileID:     "atb.profile.privileged_tool_action",
			WorkflowClass: "privileged_tool_action",
			Overall:       0.87,
			Grade:         "High",
			SubScores: map[string]float64{
				"EC": 1,
				"FC": 0.9,
				"RC": 0.8,
				"TC": 0.7,
				"SC": 0.6,
				"XC": 0.5,
				"AC": 0.4,
				"GC": 0.3,
			},
			AnchorQuality: trust.AnchorQuality{
				Label: "digest-only",
				XC:    0.5,
				AC:    0.4,
			},
		},
	}

	var out bytes.Buffer
	renderTrustReportText(&out, report)
	got := out.String()

	for _, want := range []string{
		"CAS:",
		"  Profile:   atb.profile.privileged_tool_action",
		"  Class:     privileged_tool_action",
		"  Grade:     High  (0.87)",
		"  Anchor:    digest-only  (XC=0.50 AC=0.40)",
		"  Sub-scores:",
		"    EC  1.00   FC  0.90   RC  0.80   TC  0.70",
		"    SC  0.60   XC  0.50   AC  0.40   GC  0.30",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderTrustReportText() missing %q in output:\n%s", want, got)
		}
	}
}

func TestTrustReportRenderTextIncludesCheckDetails(t *testing.T) {
	report := trust.Report{
		Status:     trust.StatusFail,
		BundlePath: "/tmp/run.atb/bundle.atb",
		Gate: trust.Gate{
			Status: trust.StatusFail,
		},
		Summary: trust.Summary{
			Total: 3,
			Pass:  1,
			Warn:  1,
			Fail:  1,
		},
		Categories: []trust.Category{
			{
				Key:    "obligation_profile",
				Status: trust.StatusFail,
				Checks: []trust.Check{
					{ID: "profile_failure_1", Title: "Profile Critical Failure", Status: trust.StatusFail, Details: "missing_event: ai.human.approval required when actions execute"},
					{ID: "profile_warning_1", Title: "Profile Required Warning", Status: trust.StatusWarn, Details: "ai.human.approval required for high-impact action types"},
				},
			},
		},
	}

	var out bytes.Buffer
	renderTrustReportText(&out, report)
	got := out.String()

	for _, want := range []string{
		"Check Details:",
		"[FAIL] obligation_profile Profile Critical Failure: missing_event: ai.human.approval required when actions execute",
		"[WARN] obligation_profile Profile Required Warning: ai.human.approval required for high-impact action types",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderTrustReportText() missing %q in output:\n%s", want, got)
		}
	}
}

func TestRunTrustReportMarkdown_AutoDetectedRAGIncludesSections(t *testing.T) {
	bundlePath := writeTrustReportRAGBundle(t, true, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runTrustReport([]string{bundlePath, "--format", "markdown"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runTrustReport() exit code = %d, want %d, stderr=%s", exitCode, exitSuccess, stderr.String())
	}

	rendered := stdout.String()
	for _, want := range []string{
		"## Model invocation",
		"- model_provider: `openai`",
		"- model_id: `gpt-4o`",
		"- model_parameters_digest present: `true`",
		"## Retrieval",
		"- ai.retrieval.executed present: `true`",
		"## Response",
		"- ai.response.sent present: `true`",
		"- request_id binding confirmed: `true`",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("runTrustReport() missing %q in markdown output:\n%s", want, rendered)
		}
	}
}

func TestRunTrustReportJSON_RAGUsesTrustReportSections(t *testing.T) {
	bundlePath := writeTrustReportRAGBundle(t, false, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runTrustReport(
		[]string{bundlePath, "--format", "json", "--profile", trustReportRAGAnswerProfileID},
		&stdout,
		&stderr,
	)
	if exitCode != exitSuccess {
		t.Fatalf("runTrustReport() exit code = %d, want %d, stderr=%s", exitCode, exitSuccess, stderr.String())
	}

	var report verifypkg.TrustReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal trust report: %v\noutput=%s", err, stdout.String())
	}

	if report.ProfileID != trustReportRAGAnswerProfileID {
		t.Fatalf("ProfileID = %q, want %q", report.ProfileID, trustReportRAGAnswerProfileID)
	}
	if report.BundlePath != bundlePath {
		t.Fatalf("BundlePath = %q, want %q", report.BundlePath, bundlePath)
	}
	if !report.Pass {
		t.Fatalf("expected trust report pass, got %+v", report)
	}
	if report.ResidualRisk == "" {
		t.Fatalf("expected residual risk, got %+v", report)
	}
	if len(report.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(report.Sections))
	}

	wantSections := []struct {
		title string
		note  string
	}{
		{title: "Model invocation", note: "model_parameters_digest present"},
		{title: "Retrieval", note: "ai.retrieval.executed absent: retrieval step not recorded"},
		{title: "Response", note: "ai.response.sent absent"},
	}
	for _, want := range wantSections {
		section := trustReportSectionByTitle(t, report.Sections, want.title)
		if !containsTrustReportString(section.Notes, want.note) {
			t.Fatalf("expected note %q in section %+v", want.note, section)
		}
	}
}

func TestRunTrustReportJSON_NoProfileReturnsExitOne(t *testing.T) {
	bundlePath := filepath.Join(t.TempDir(), "run.atb", "bundle.atb")
	b := newTestBundle(t)
	appendTestBundleEventWithOptions(t, b, "agent.run", map[string]any{
		"workflow": "incident-review",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:00:00Z"})
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runTrustReport([]string{bundlePath, "--format", "json"}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("runTrustReport() exit code = %d, want %d, stderr=%s", exitCode, exitUserError, stderr.String())
	}

	var report verifypkg.TrustReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal trust report: %v\noutput=%s", err, stdout.String())
	}
	if report.Pass {
		t.Fatalf("expected trust report pass=false, got %+v", report)
	}
	if report.ProfileID != "" {
		t.Fatalf("expected empty profile ID, got %+v", report)
	}
	if report.Chain.RecordCount != len(b.Records) {
		t.Fatalf("unexpected record count: got %d want %d", report.Chain.RecordCount, len(b.Records))
	}
	if len(report.Sections) != 0 {
		t.Fatalf("expected no sections, got %+v", report.Sections)
	}
}

func trustReportSectionByTitle(t testing.TB, sections []verifypkg.TrustSection, title string) verifypkg.TrustSection {
	t.Helper()
	for _, section := range sections {
		if section.Title == title {
			return section
		}
	}
	t.Fatalf("section %q not found in %+v", title, sections)
	return verifypkg.TrustSection{}
}

func writeTrustReportRAGBundle(t testing.TB, includeRetrieval bool, includeResponse bool) string {
	t.Helper()

	bundlePath := filepath.Join(t.TempDir(), "run.atb", "bundle.atb")
	b := newTestBundle(t)

	appendTestBundleEventWithOptions(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "rag_answer",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:00:00Z"})
	if includeRetrieval {
		appendTestBundleEventWithOptions(t, b, event.TypeAIRetrievalExecuted, map[string]any{
			"retrieval_query_hash":     "query-hash",
			"retrieval_corpus_id":      "corpus-1",
			"retrieval_corpus_version": "v7",
			"top_k":                    5,
			"result_set_digest":        "results-digest",
		}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:01:00Z"})
	}
	appendTestBundleEventWithOptions(t, b, event.TypeAIModelInvoked, map[string]any{
		"model_provider":          "openai",
		"model_id":                "gpt-4o",
		"model_parameters_digest": "sha256-params-def",
		"prompt_digest":           "sha256-prompt-ghi",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:02:00Z"})
	appendTestBundleEventWithOptions(t, b, event.TypeAIModelOutput, map[string]any{
		"output_digest": "sha256-output-jkl",
		"output_format": "text/plain",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:03:00Z"})
	if includeResponse {
		appendTestBundleEventWithOptions(t, b, event.TypeAIResponseSent, map[string]any{
			"request_id":    "req-1",
			"output_digest": "sha256-output-jkl",
		}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:04:00Z"})
	}

	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return bundlePath
}

func containsTrustReportString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
