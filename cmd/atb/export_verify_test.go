package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestRunExportWithVerifyScenarios(t *testing.T) {
	tests := []struct {
		name                   string
		withVerify             bool
		jsonOutput             bool
		profilePass            bool
		wantExitCode           int
		wantSidecar            bool
		wantVerificationInJSON bool
		wantStatus             string
	}{
		{
			name:         "with verify writes sidecar and exits success",
			withVerify:   true,
			profilePass:  true,
			wantExitCode: exitSuccess,
			wantSidecar:  true,
			wantStatus:   "PASS",
		},
		{
			name:         "with verify returns profile failure exit code",
			withVerify:   true,
			profilePass:  false,
			wantExitCode: exitVerifyFailure,
			wantSidecar:  true,
			wantStatus:   "FAIL",
		},
		{
			name:         "without verify does not write sidecar",
			withVerify:   false,
			profilePass:  true,
			wantExitCode: exitSuccess,
			wantSidecar:  false,
		},
		{
			name:                   "with verify and json appends verification summary",
			withVerify:             true,
			jsonOutput:             true,
			profilePass:            true,
			wantExitCode:           exitSuccess,
			wantSidecar:            true,
			wantVerificationInJSON: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withTempCWD(t, func(tmp string) {
				preparePhase4Docs(t)
				writeExportVerifyBundle(t, bundle.DefaultPath(), tc.profilePass)

				outputPath := filepath.Join("exports", "soc2.zip")
				args := []string{
					"--format", "soc2",
					"--bundle", bundle.DefaultPath(),
					"--output", outputPath,
				}
				if tc.withVerify {
					args = append(args, "--with-verify")
				}
				if tc.jsonOutput {
					args = append(args, "--json")
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

					var report verifypkg.Report
					if err := json.Unmarshal(sidecarBytes, &report); err != nil {
						t.Fatalf("unmarshal sidecar report: %v", err)
					}
					if report.BundlePath != bundle.DefaultPath() {
						t.Fatalf("unexpected report bundle path: got %q want %q", report.BundlePath, bundle.DefaultPath())
					}
				} else if _, err := os.Stat(sidecarAbsPath); !os.IsNotExist(err) {
					t.Fatalf("expected no sidecar at %s, got err=%v", sidecarAbsPath, err)
				}

				if tc.jsonOutput {
					var payload map[string]any
					if err := json.Unmarshal([]byte(result.stdout), &payload); err != nil {
						t.Fatalf("unmarshal export json output: %v", err)
					}

					verificationValue, hasVerification := payload["verification"]
					if tc.wantVerificationInJSON != hasVerification {
						t.Fatalf("unexpected verification key presence: got %t want %t", hasVerification, tc.wantVerificationInJSON)
					}
					if tc.wantVerificationInJSON {
						verification, ok := verificationValue.(map[string]any)
						if !ok {
							t.Fatalf("verification payload has unexpected type %T", verificationValue)
						}
						if got := verification["sidecar"]; got != sidecarPath {
							t.Fatalf("unexpected verification sidecar: got %v want %q", got, sidecarPath)
						}
					}
					return
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

	b := bundle.New()
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
		appendTestBundleEventWithOptions(t, b, event.TypeAIHumanApproval, map[string]any{
			"approval_id":          "appr-1",
			"approver_id_hash":     "approver-hash",
			"approval_outcome":     "approve",
			"justification_digest": "just-digest",
			"action_id":            "act-1",
		}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:03:00Z"})
		appendTestBundleEventWithOptions(t, b, event.TypeAIActionExecuted, map[string]any{
			"action_id":           "act-1",
			"execution_outcome":   "success",
			"tool_receipt_digest": "tool-digest",
		}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:05:00Z"})
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
