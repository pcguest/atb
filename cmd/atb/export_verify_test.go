// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestRunExportWithVerifyScenarios(t *testing.T) {
	tests := []struct {
		name         string
		withVerify   bool
		profilePass  bool
		stdoutMode   bool
		wantExitCode int
		wantSidecar  bool
		wantWarning  string
		wantStatus   string
		checkReport  func(*testing.T, verifypkg.Report)
	}{
		{
			name:         "with_verify_writes_sidecar",
			withVerify:   true,
			profilePass:  true,
			wantExitCode: exitSuccess,
			wantSidecar:  true,
			wantStatus:   "PASS",
		},
		{
			name:         "sidecar_json_is_complete",
			withVerify:   true,
			profilePass:  true,
			wantExitCode: exitSuccess,
			wantSidecar:  true,
			wantStatus:   "PASS",
			checkReport: func(t *testing.T, report verifypkg.Report) {
				t.Helper()
				if len(report.Profiles) == 0 {
					t.Fatalf("expected at least one profile result in sidecar")
				}
				if report.Profiles[0].ProfileID == "" {
					t.Fatalf("expected non-empty profile ID in sidecar")
				}
			},
		},
		{
			name:         "passing_bundle_exits_zero",
			withVerify:   true,
			profilePass:  true,
			wantExitCode: exitSuccess,
			wantSidecar:  true,
			wantStatus:   "PASS",
		},
		{
			name:         "failing_bundle_exits_profile_failure",
			withVerify:   true,
			profilePass:  false,
			wantExitCode: exitVerifyFailure,
			wantSidecar:  true,
			wantStatus:   "FAIL",
		},
		{
			name:         "without_flag_no_sidecar",
			withVerify:   false,
			profilePass:  true,
			wantExitCode: exitSuccess,
			wantSidecar:  false,
		},
		{
			name:         "stdout_write_no_sidecar_no_error",
			withVerify:   true,
			profilePass:  true,
			stdoutMode:   true,
			wantExitCode: exitSuccess,
			wantSidecar:  false,
			wantWarning:  "warning: --with-verify ignored when writing to stdout\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withTempCWD(t, func(tmp string) {
				preparePhase4Docs(t)
				writeExportVerifyBundle(t, bundle.DefaultPath(), tc.profilePass)

				if tc.stdoutMode {
					cfg := exportConfig{
						Format:     exportFormatSOC2,
						BundlePath: bundle.DefaultPath(),
						WithVerify: true,
					}
					result, err := buildExport(time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC), cfg)
					if err != nil {
						t.Fatalf("build export: %v", err)
					}

					var stderr bytes.Buffer
					_, summary, err := writeExportVerificationSidecar(cfg, result, &stderr)
					if err != nil {
						t.Fatalf("writeExportVerificationSidecar: %v", err)
					}
					gotExitCode := exitSuccess
					if gotExitCode != tc.wantExitCode {
						t.Fatalf("unexpected exit code: got %d want %d", gotExitCode, tc.wantExitCode)
					}
					if summary != nil {
						t.Fatalf("expected nil verification summary when sidecar writing is skipped, got %+v", summary)
					}
					if stderr.String() != tc.wantWarning {
						t.Fatalf("unexpected stderr warning: got %q want %q", stderr.String(), tc.wantWarning)
					}

					sidecars, err := filepath.Glob(filepath.Join(tmp, "*.verify.json"))
					if err != nil {
						t.Fatalf("glob sidecars: %v", err)
					}
					if len(sidecars) != 0 {
						t.Fatalf("expected no sidecar files, found %v", sidecars)
					}
					return
				}

				outputPath := filepath.Join("exports", "soc2.zip")
				args := []string{
					"--format", "soc2",
					"--bundle", bundle.DefaultPath(),
					"--output", outputPath,
				}
				if tc.withVerify {
					args = append(args, "--with-verify")
				}

				result := captureExportRun(t, args)
				if result.exitCode != tc.wantExitCode {
					t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", result.exitCode, tc.wantExitCode, result.stderr)
				}
				if result.stderr != "" {
					t.Fatalf("unexpected stderr output: %q", result.stderr)
				}

				sidecarPath := outputPath + ".verify.json"
				sidecarAbsPath := filepath.Join(tmp, sidecarPath)
				if tc.wantSidecar {
					sidecarBytes, err := os.ReadFile(sidecarAbsPath)
					if err != nil {
						t.Fatalf("read sidecar: %v", err)
					}
					info, err := os.Stat(sidecarAbsPath)
					if err != nil {
						t.Fatalf("stat sidecar: %v", err)
					}
					if got := info.Mode().Perm(); got != 0o600 {
						t.Fatalf("sidecar mode=%#o, want 0600", got)
					}

					var sidecar verifypkg.ExportVerificationSidecar
					if err := json.Unmarshal(sidecarBytes, &sidecar); err != nil {
						t.Fatalf("unmarshal sidecar: %v", err)
					}
					report := sidecar.Report
					if report.BundlePath != bundle.DefaultPath() {
						t.Fatalf("unexpected report bundle path: got %q want %q", report.BundlePath, bundle.DefaultPath())
					}
					if sidecar.ProvabilityLayers == nil {
						t.Fatalf("expected provability_layers in sidecar")
					}
					if tc.checkReport != nil {
						tc.checkReport(t, report)
					}
				} else if _, err := os.Stat(sidecarAbsPath); !os.IsNotExist(err) {
					t.Fatalf("expected no sidecar at %s, got err=%v", sidecarAbsPath, err)
				}

				if tc.withVerify {
					if !strings.Contains(result.stdout, "Verification: "+tc.wantStatus) {
						t.Fatalf("expected verification status line in stdout, got %q", result.stdout)
					}
					if !strings.Contains(result.stdout, "Sidecar written: "+sidecarPath) {
						t.Fatalf("expected sidecar line in stdout, got %q", result.stdout)
					}
					return
				}

				if strings.Contains(result.stdout, "Sidecar written:") {
					t.Fatalf("did not expect sidecar output when --with-verify is absent: %q", result.stdout)
				}
			})
		})
	}
}

func captureExportRun(t *testing.T, args []string) struct {
	stdout   string
	stderr   string
	exitCode int
} {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runExport(args, &stdout, &stderr)
	return struct {
		stdout   string
		stderr   string
		exitCode int
	}{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitCode,
	}
}

func writeExportVerifyBundle(t testing.TB, path string, profilePass bool) {
	t.Helper()

	b := newTestBundle(t)
	appendTestBundleEventWithOptions(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "approve-change",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:00:00Z"})
	appendTestBundleEventWithOptions(t, b, event.TypeAIActionPrecommit, map[string]any{
		"action_id":                "act-1",
		"action_type":              "deploy_change",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "svc-1",
		"intended_effect":          "deploy build 42",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:01:00Z"})
	appendTestBundleEventWithOptions(t, b, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:02:00Z"})

	if profilePass {
		appendTestBundleEventWithOptions(t, b, event.TypeAIActionExecuted, map[string]any{
			"action_id":           "act-1",
			"execution_outcome":   "success",
			"tool_receipt_digest": "tool-digest",
		}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:05:00Z"})
		appendTestBundleEventWithOptions(t, b, event.TypeAIHumanApproval, map[string]any{
			"approval_id":          "appr-1",
			"approver_id_hash":     "approver-hash",
			"approval_outcome":     "approve",
			"justification_digest": "just-digest",
			"action_id":            "act-1",
		}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:05:30Z"})
		appendTestBundleEventWithOptions(t, b, event.TypeAIActionCommitted, map[string]any{
			"action_id":           "act-1",
			"commit_outcome":      "success",
			"sink_receipt_digest": "sink-digest",
		}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:06:00Z"})
	}

	if err := b.Save(path); err != nil {
		t.Fatalf("save export verify bundle: %v", err)
	}
}
