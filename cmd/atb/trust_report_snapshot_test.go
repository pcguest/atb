// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

type snapshotAppend struct {
	eventType string
	data      any
}

type snapshotProfileCase struct {
	name                    string
	profileID               string
	minRecordCount          int
	expectCAS               bool
	events                  []snapshotAppend
	trustReportArgs         []string
	verifyArgs              []string
	wantTrustReportExitCode int
	wantVerifyExitCode      int
	afterBuild              func(t testing.TB, bundlePath string)
	assertTrustReport       func(t *testing.T, report verifypkg.TrustReport)
	assertVerifyReport      func(t *testing.T, report verifypkg.VerifierReport)
}

var snapshotWorkingDirMu sync.Mutex

func TestTrustReportJSONSnapshot(t *testing.T) {
	for _, tc := range trustReportSnapshotCases() {
		tc := tc
		name := tc.name
		if name == "" {
			name = tc.profileID
		}

		t.Run(name, func(t *testing.T) {
			bundlePath := buildSnapshotBundle(t, tc.events)
			if tc.afterBuild != nil {
				tc.afterBuild(t, bundlePath)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			args := append([]string{bundlePath, "--format", "json"}, tc.trustReportArgs...)
			exitCode := runTrustReport(args, &stdout, &stderr)
			if exitCode != tc.wantTrustReportExitCode {
				t.Errorf("runTrustReport() exit code = %d, want %d (stderr=%q)", exitCode, tc.wantTrustReportExitCode, stderr.String())
			}

			var report verifypkg.TrustReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Errorf("unmarshal trust report: %v\noutput=%s", err, stdout.String())
				return
			}

			if tc.assertTrustReport != nil {
				tc.assertTrustReport(t, report)
				return
			}

			assertDefaultTrustReportSnapshot(t, tc, report)
		})
	}
}

func snapshotProfileCases() []snapshotProfileCase {
	return []snapshotProfileCase{
		{
			name:                    "atb.profile.rag_answer",
			profileID:               "atb.profile.rag_answer",
			minRecordCount:          3,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitSuccess,
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-rag",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "rag_answer",
					},
				},
				{
					eventType: event.TypeAIModelInvoked,
					data: map[string]any{
						"model_provider":          "openai",
						"model_id":                "gpt-4o",
						"model_parameters_digest": "params-digest",
						"prompt_digest":           "prompt-digest",
					},
				},
				{
					eventType: event.TypeAIModelOutput,
					data: map[string]any{
						"output_digest": "output-digest",
						"output_format": "text/plain",
					},
				},
			},
		},
		{
			name:                    "atb.profile.privileged_tool_action",
			profileID:               "atb.profile.privileged_tool_action",
			minRecordCount:          6,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitSuccess,
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-privileged",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "privileged_tool_action",
					},
				},
				{
					eventType: event.TypeAIPolicyDecision,
					data: map[string]any{
						"policy_id":             "pol-1",
						"policy_version":        "2026-03",
						"decision":              "allow",
						"action_id":             "act-1",
						"subject_id_hash":       "subject-hash",
						"decision_reason_codes": []string{"approved"},
					},
				},
				{
					eventType: event.TypeAIActionPrecommit,
					data: map[string]any{
						"action_id":                "act-1",
						"action_type":              "deploy_change",
						"action_parameters_digest": "params-digest",
						"target_resource_id":       "svc-1",
						"intended_effect":          "deploy build 42",
					},
				},
				{
					eventType: event.TypeAIActionExecuted,
					data: map[string]any{
						"action_id":           "act-1",
						"execution_outcome":   "success",
						"tool_receipt_digest": "tool-digest",
					},
				},
				{
					eventType: event.TypeAIHumanApproval,
					data: map[string]any{
						"approval_id":          "appr-1",
						"approver_id_hash":     "approver-hash",
						"approval_outcome":     "approved",
						"justification_digest": "just-digest",
						"action_id":            "act-1",
					},
				},
				{
					eventType: event.TypeAIActionCommitted,
					data: map[string]any{
						"action_id":           "act-1",
						"commit_outcome":      "success",
						"sink_receipt_digest": "sink-digest",
					},
				},
			},
		},
		{
			name:                    "atb.profile.data_export",
			profileID:               "atb.profile.data_export",
			minRecordCount:          6,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitSuccess,
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-export",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "data_export",
					},
				},
				{
					eventType: event.TypeAIPolicyDecision,
					data: map[string]any{
						"policy_id":             "pol-1",
						"policy_version":        "2026-03",
						"decision":              "allow",
						"action_id":             "act-1",
						"subject_id_hash":       "subject-hash",
						"decision_reason_codes": []string{"export_allowed"},
					},
				},
				{
					eventType: event.TypeDataExportPrecommit,
					data: map[string]any{
						"action_id":                "act-1",
						"action_type":              "export_data",
						"action_parameters_digest": "params-digest",
						"target_resource_id":       "dataset-1",
						"intended_effect":          "export approved dataset",
					},
				},
				{
					eventType: event.TypeDataExportExecuted,
					data: map[string]any{
						"action_id":           "act-1",
						"execution_outcome":   "success",
						"tool_receipt_digest": "tool-digest",
					},
				},
				{
					eventType: event.TypeAIHumanApproval,
					data: map[string]any{
						"approval_id":          "appr-1",
						"approver_id_hash":     "approver-hash",
						"approval_outcome":     "approved",
						"justification_digest": "just-digest",
						"action_id":            "act-1",
					},
				},
			},
		},
		{
			name:                    "atb.profile.background_automation",
			profileID:               "atb.profile.background_automation",
			minRecordCount:          4,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitSuccess,
			events: []snapshotAppend{
				{
					eventType: event.TypeAIJobScheduled,
					data: map[string]any{
						"job_id":               "job-1",
						"job_type":             "nightly_sync",
						"trigger_source":       "cron",
						"scheduled_by_id_hash": "scheduler-hash",
					},
				},
				{
					eventType: event.TypeAIJobStarted,
					data: map[string]any{
						"job_id":         "job-1",
						"worker_id_hash": "worker-hash",
						"started_at":     "2026-03-27T12:02:00Z",
					},
				},
				{
					eventType: event.TypeAIJobCompleted,
					data: map[string]any{
						"job_id":            "job-1",
						"outcome":           "success",
						"completion_reason": "completed",
					},
				},
			},
		},
		{
			name:                    "atb.profile.policy_decision",
			profileID:               "atb.profile.policy_decision",
			minRecordCount:          4,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitSuccess,
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-policy",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "policy_decision",
					},
				},
				{
					eventType: event.TypeAIActionPrecommit,
					data: map[string]any{
						"action_id":                "act-1",
						"action_type":              "policy_decision",
						"action_parameters_digest": "params-digest",
						"target_resource_id":       "resource-1",
						"intended_effect":          "record decision context",
					},
				},
				{
					eventType: event.TypeAIPolicyDecision,
					data: map[string]any{
						"policy_id":             "pol-1",
						"policy_version":        "2026-03",
						"decision":              "allow",
						"action_id":             "act-1",
						"subject_id_hash":       "subject-hash",
						"decision_reason_codes": []string{"approved"},
					},
				},
			},
		},
		{
			name:                    "atb.profile.human_override",
			profileID:               "atb.profile.human_override",
			minRecordCount:          6,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitSuccess,
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-override",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "human_override",
					},
				},
				{
					eventType: event.TypeAIHumanApproval,
					data: map[string]any{
						"approval_id":          "appr-1",
						"approver_id_hash":     "approver-hash",
						"approval_outcome":     "approved",
						"justification_digest": "just-digest",
						"action_id":            "act-1",
					},
				},
				{
					eventType: event.TypeAIActionPrecommit,
					data: map[string]any{
						"action_id":                "act-1",
						"action_type":              "override_action",
						"action_parameters_digest": "params-digest",
						"target_resource_id":       "svc-1",
						"intended_effect":          "run approved override",
					},
				},
				{
					eventType: event.TypeAIActionExecuted,
					data: map[string]any{
						"action_id":           "act-1",
						"execution_outcome":   "success",
						"tool_receipt_digest": "tool-digest",
					},
				},
				{
					eventType: event.TypeAIActionCommitted,
					data: map[string]any{
						"action_id":           "act-1",
						"commit_outcome":      "success",
						"sink_receipt_digest": "sink-digest",
					},
				},
			},
		},
	}
}

func trustReportSnapshotCases() []snapshotProfileCase {
	cases := append([]snapshotProfileCase{}, snapshotProfileCases()...)
	return append(cases,
		snapshotProfileCase{
			name:                    "empty_bundle",
			minRecordCount:          1,
			wantTrustReportExitCode: exitUserError,
			wantVerifyExitCode:      exitSuccess,
			assertTrustReport: func(t *testing.T, report verifypkg.TrustReport) {
				t.Helper()

				if report.BundlePath == "" {
					t.Errorf("BundlePath is empty")
				}
				if report.ProfileID != "" {
					t.Errorf("ProfileID = %q, want empty", report.ProfileID)
				}
				if report.ResidualRisk == "" {
					t.Errorf("ResidualRisk is empty")
				}
				if !report.Chain.Valid {
					t.Errorf("Chain.Valid = false, want true")
				}
				if report.Chain.RecordCount != 1 {
					t.Errorf("Chain.RecordCount = %d, want 1", report.Chain.RecordCount)
				}
				if len(report.Sections) == 0 {
					return
				}
				if len(report.Sections) != 1 || report.Sections[0].Title != "Evidence summary" {
					t.Errorf("Sections = %+v, want empty or a single Evidence summary section", report.Sections)
				}
			},
		},
		snapshotProfileCase{
			name:                    "rag_answer_missing_model_invoked",
			profileID:               "atb.profile.rag_answer",
			minRecordCount:          2,
			wantTrustReportExitCode: exitUserError,
			wantVerifyExitCode:      exitVerifyFailure,
			trustReportArgs:         []string{"--profile", "atb.profile.rag_answer"},
			verifyArgs:              []string{"--profile", "atb.profile.rag_answer"},
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-rag-missing-model",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "rag_answer",
					},
				},
			},
			assertTrustReport: func(t *testing.T, report verifypkg.TrustReport) {
				t.Helper()

				if report.BundlePath == "" {
					t.Errorf("BundlePath is empty")
				}
				if report.ProfileID != "atb.profile.rag_answer" {
					t.Errorf("ProfileID = %q, want %q", report.ProfileID, "atb.profile.rag_answer")
				}
				if report.Pass {
					t.Errorf("Pass = true, want false")
				}
				if report.ResidualRisk == "" {
					t.Errorf("ResidualRisk is empty")
				}
				if len(report.Sections) < 2 {
					t.Errorf("len(Sections) = %d, want >= 2", len(report.Sections))
				}
				section := snapshotTrustReportSectionByTitle(report, "Model invocation")
				if section == nil {
					t.Fatalf("Model invocation section missing")
				}
				if section.Pass {
					t.Errorf("Model invocation section Pass = true, want false")
				}
			},
		},
		snapshotProfileCase{
			name:                    "background_automation_missing_completed",
			profileID:               "atb.profile.background_automation",
			minRecordCount:          3,
			wantTrustReportExitCode: exitUserError,
			wantVerifyExitCode:      exitVerifyFailure,
			trustReportArgs:         []string{"--profile", "atb.profile.background_automation"},
			verifyArgs:              []string{"--profile", "atb.profile.background_automation"},
			events: []snapshotAppend{
				{
					eventType: event.TypeAIJobScheduled,
					data: map[string]any{
						"job_id":               "job-missing-completed",
						"job_type":             "nightly_sync",
						"trigger_source":       "cron",
						"scheduled_by_id_hash": "scheduler-hash",
					},
				},
				{
					eventType: event.TypeAIJobStarted,
					data: map[string]any{
						"job_id":         "job-missing-completed",
						"worker_id_hash": "worker-hash",
						"started_at":     "2026-03-27T12:02:00Z",
					},
				},
			},
			assertTrustReport: func(t *testing.T, report verifypkg.TrustReport) {
				t.Helper()

				if report.BundlePath == "" {
					t.Errorf("BundlePath is empty")
				}
				if !report.Chain.Valid {
					t.Errorf("Chain.Valid = false, want true")
				}
				if report.Pass {
					t.Errorf("Pass = true, want false")
				}
				if report.ResidualRisk == "" {
					t.Errorf("ResidualRisk is empty")
				}
				section := snapshotTrustReportSectionByTitle(report, "Job execution")
				if section == nil {
					t.Fatalf("Job execution section missing")
				}
				if section.Pass {
					t.Errorf("Job execution section Pass = true, want false")
				}
			},
		},
		snapshotProfileCase{
			name:                    "privileged_tool_action_deny_decision",
			profileID:               "atb.profile.privileged_tool_action",
			minRecordCount:          6,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitSuccess,
			trustReportArgs:         []string{"--profile", "atb.profile.privileged_tool_action"},
			verifyArgs:              []string{"--profile", "atb.profile.privileged_tool_action"},
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-privileged-deny",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "privileged_tool_action",
					},
				},
				{
					eventType: event.TypeAIPolicyDecision,
					data: map[string]any{
						"policy_id":             "pol-1",
						"policy_version":        "2026-03",
						"decision":              "deny",
						"action_id":             "act-1",
						"subject_id_hash":       "subject-hash",
						"decision_reason_codes": []string{"blocked"},
					},
				},
				{
					eventType: event.TypeAIActionPrecommit,
					data: map[string]any{
						"action_id":                "act-1",
						"action_type":              "deploy_change",
						"action_parameters_digest": "params-digest",
						"target_resource_id":       "svc-1",
						"intended_effect":          "deploy build 42",
					},
				},
				{
					eventType: event.TypeAIActionExecuted,
					data: map[string]any{
						"action_id":           "act-1",
						"execution_outcome":   "success",
						"tool_receipt_digest": "tool-digest",
					},
				},
				{
					eventType: event.TypeAIHumanApproval,
					data: map[string]any{
						"approval_id":          "appr-1",
						"approver_id_hash":     "approver-hash",
						"approval_outcome":     "approved",
						"justification_digest": "just-digest",
						"action_id":            "act-1",
					},
				},
				{
					eventType: event.TypeAIActionCommitted,
					data: map[string]any{
						"action_id":           "act-1",
						"commit_outcome":      "success",
						"sink_receipt_digest": "sink-digest",
					},
				},
			},
			assertTrustReport: func(t *testing.T, report verifypkg.TrustReport) {
				t.Helper()

				assertDefaultTrustReportSnapshot(t, snapshotProfileCase{
					profileID:      "atb.profile.privileged_tool_action",
					minRecordCount: 6,
					expectCAS:      true,
				}, report)
				if !report.Pass {
					t.Errorf("Pass = false, want true")
				}
				section := snapshotTrustReportSectionByTitle(report, "Policy decision")
				if section == nil {
					t.Fatalf("Policy decision section missing")
				}
				if got := section.Fields["decision"]; got != "deny" {
					t.Errorf("Policy decision fields[decision] = %q, want %q", got, "deny")
				}
			},
		},
	)
}

func verifySnapshotCases() []snapshotProfileCase {
	cases := append([]snapshotProfileCase{}, snapshotProfileCases()...)
	return append(cases,
		snapshotProfileCase{
			name:                    "empty_bundle_no_profile",
			minRecordCount:          1,
			wantTrustReportExitCode: exitUserError,
			wantVerifyExitCode:      exitSuccess,
			assertVerifyReport: func(t *testing.T, report verifypkg.VerifierReport) {
				t.Helper()

				if report.BundlePath == "" {
					t.Errorf("BundlePath is empty")
				}
				if report.ProfileID != "" {
					t.Errorf("ProfileID = %q, want empty", report.ProfileID)
				}
				if report.ResidualRisk == "" {
					t.Errorf("ResidualRisk is empty")
				}
				if len(report.Failures) != 0 {
					t.Errorf("Failures = %+v, want none", report.Failures)
				}
			},
		},
		snapshotProfileCase{
			name:                    "rag_answer_chain_integrity",
			profileID:               "atb.profile.rag_answer",
			minRecordCount:          3,
			expectCAS:               true,
			wantTrustReportExitCode: exitSuccess,
			wantVerifyExitCode:      exitIntegrityFailure,
			trustReportArgs:         []string{"--profile", "atb.profile.rag_answer"},
			verifyArgs:              []string{"--profile", "atb.profile.rag_answer"},
			events: []snapshotAppend{
				{
					eventType: event.TypeAIRequestReceived,
					data: map[string]any{
						"request_id":    "req-rag-tampered",
						"actor_id_hash": "actor-hash",
						"purpose_tag":   "rag_answer",
					},
				},
				{
					eventType: event.TypeAIModelInvoked,
					data: map[string]any{
						"model_provider":          "openai",
						"model_id":                "gpt-4o",
						"model_parameters_digest": "params-digest",
						"prompt_digest":           "prompt-digest",
					},
				},
				{
					eventType: event.TypeAIModelOutput,
					data: map[string]any{
						"output_digest": "output-digest",
						"output_format": "text/plain",
					},
				},
			},
			afterBuild: func(t testing.TB, bundlePath string) {
				t.Helper()
				tamperSnapshotBundleRecordData(t, bundlePath, 1, "tampered", true)
			},
			assertVerifyReport: func(t *testing.T, report verifypkg.VerifierReport) {
				t.Helper()

				if report.BundlePath == "" {
					t.Errorf("BundlePath is empty")
				}
				if report.ProfileID != "atb.profile.rag_answer" {
					t.Errorf("ProfileID = %q, want %q", report.ProfileID, "atb.profile.rag_answer")
				}
				if report.Pass {
					t.Errorf("Pass = true, want false")
				}
				if report.CASGrade != "Insufficient" {
					t.Errorf("CASGrade = %q, want %q", report.CASGrade, "Insufficient")
				}
				if len(report.SubScores) == 0 {
					t.Errorf("SubScores is empty, want diagnostic CAS output")
				}
				if report.ResidualRisk != "Critical" {
					t.Errorf("ResidualRisk = %q, want %q", report.ResidualRisk, "Critical")
				}
			},
		},
	)
}

func assertDefaultTrustReportSnapshot(t *testing.T, tc snapshotProfileCase, report verifypkg.TrustReport) {
	t.Helper()

	if report.BundlePath == "" {
		t.Errorf("BundlePath is empty")
	}
	if report.ProfileID != tc.profileID {
		t.Errorf("ProfileID = %q, want %q", report.ProfileID, tc.profileID)
	}
	if report.ResidualRisk == "" {
		t.Errorf("ResidualRisk is empty")
	}
	if len(report.Sections) == 0 {
		t.Errorf("Sections is empty")
	}
	if !report.Chain.Valid {
		t.Errorf("Chain.Valid = false, want true")
	}
	if report.Chain.RecordCount < tc.minRecordCount {
		t.Errorf("Chain.RecordCount = %d, want >= %d", report.Chain.RecordCount, tc.minRecordCount)
	}
	if tc.expectCAS && report.CASGrade == "" {
		t.Errorf("CASGrade is empty")
	}
	if tc.expectCAS && report.CAS == nil {
		t.Errorf("CAS object is nil, want cas block in JSON output")
	}
}

func snapshotTrustReportSectionByTitle(report verifypkg.TrustReport, title string) *verifypkg.TrustSection {
	for i := range report.Sections {
		if report.Sections[i].Title == title {
			return &report.Sections[i]
		}
	}
	return nil
}

func tamperSnapshotBundleRecordData(t testing.TB, bundlePath string, sequence int, key string, value any) {
	t.Helper()

	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle %s: %v", bundlePath, err)
	}

	lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	found := false
	for i, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("unmarshal bundle record %d: %v", i, err)
		}

		eventData, ok := record["event"].(map[string]any)
		if !ok {
			t.Fatalf("bundle record %d event has type %T, want object", i, record["event"])
		}

		seq, ok := eventData["seq"].(float64)
		if !ok {
			t.Fatalf("bundle record %d seq has type %T, want number", i, eventData["seq"])
		}
		if int(seq) != sequence {
			continue
		}

		fields, ok := eventData["data"].(map[string]any)
		if !ok {
			t.Fatalf("bundle record %d data has type %T, want object", i, eventData["data"])
		}
		fields[key] = value

		encoded, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal tampered record %d: %v", i, err)
		}
		lines[i] = encoded
		found = true
		break
	}

	if !found {
		t.Fatalf("record with seq %d not found in %s", sequence, bundlePath)
	}

	data = append(bytes.Join(lines, []byte("\n")), '\n')
	if err := os.WriteFile(bundlePath, data, 0600); err != nil {
		t.Fatalf("rewrite tampered bundle %s: %v", bundlePath, err)
	}
}

func buildSnapshotBundle(t testing.TB, events []snapshotAppend) string {
	t.Helper()

	workDir := t.TempDir()
	withSnapshotWorkingDir(t, workDir, func() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		if exitCode := runInit(nil, &stdout, &stderr); exitCode != exitSuccess {
			t.Fatalf("runInit() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
		}

		for _, appendEvent := range events {
			payload, err := json.Marshal(appendEvent.data)
			if err != nil {
				t.Fatalf("marshal %s payload: %v", appendEvent.eventType, err)
			}

			stdout.Reset()
			stderr.Reset()
			if exitCode := runAppend([]string{appendEvent.eventType, string(payload)}, &stdout, &stderr); exitCode != exitSuccess {
				t.Fatalf("runAppend(%q) exit code = %d, want %d (stderr=%q)", appendEvent.eventType, exitCode, exitSuccess, stderr.String())
			}
		}
	})

	return filepath.Join(workDir, bundle.BundleDir, bundle.BundleFile)
}

func withSnapshotWorkingDir(t testing.TB, dir string, fn func()) {
	t.Helper()

	snapshotWorkingDirMu.Lock()
	defer snapshotWorkingDirMu.Unlock()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %q: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working directory %q: %v", wd, err)
		}
	}()

	fn()
}
