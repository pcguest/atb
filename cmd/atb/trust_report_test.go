package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/trust"
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
