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
	data      map[string]any
}

type snapshotProfileCase struct {
	profileID      string
	minRecordCount int
	expectCAS      bool
	events         []snapshotAppend
}

var snapshotWorkingDirMu sync.Mutex

func TestTrustReportJSONSnapshot(t *testing.T) {
	for _, tc := range snapshotProfileCases() {
		tc := tc
		t.Run(tc.profileID, func(t *testing.T) {
			bundlePath := buildSnapshotBundle(t, tc.events)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runTrustReport([]string{bundlePath, "--format", "json"}, &stdout, &stderr)
			if exitCode != exitSuccess {
				t.Errorf("runTrustReport() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
			}

			var report verifypkg.TrustReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Errorf("unmarshal trust report: %v\noutput=%s", err, stdout.String())
				return
			}

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
		})
	}
}

func snapshotProfileCases() []snapshotProfileCase {
	return []snapshotProfileCase{
		{
			profileID:      "atb.profile.rag_answer",
			minRecordCount: 3,
			expectCAS:      true,
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
			profileID:      "atb.profile.privileged_tool_action",
			minRecordCount: 6,
			expectCAS:      true,
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
			profileID:      "atb.profile.data_export",
			minRecordCount: 6,
			expectCAS:      false,
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
					eventType: event.TypeAIActionPrecommit,
					data: map[string]any{
						"action_id":                "act-1",
						"action_type":              "export_data",
						"action_parameters_digest": "params-digest",
						"target_resource_id":       "dataset-1",
						"intended_effect":          "export approved dataset",
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
			profileID:      "atb.profile.background_automation",
			minRecordCount: 4,
			expectCAS:      false,
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
			profileID:      "atb.profile.policy_decision",
			minRecordCount: 3,
			expectCAS:      false,
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
			profileID:      "atb.profile.human_override",
			minRecordCount: 6,
			expectCAS:      false,
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
