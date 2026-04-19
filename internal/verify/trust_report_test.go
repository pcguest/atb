package verify

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
)

func TestTrustReportFromVerify_RAGAnswer(t *testing.T) {
	b := newRAGAnswerBundle(t)

	report := Verify(b, "bundle.atb", profileIDRAGAnswer)
	trustReport := TrustReportFromVerify(report, b)

	if trustReport.BundlePath != "bundle.atb" {
		t.Fatalf("unexpected bundle path: got %q want %q", trustReport.BundlePath, "bundle.atb")
	}
	if trustReport.ProfileID != profileIDRAGAnswer {
		t.Fatalf("unexpected profile ID: got %q want %q", trustReport.ProfileID, profileIDRAGAnswer)
	}
	if trustReport.WorkflowClass != "rag_answer" {
		t.Fatalf("unexpected workflow class: got %q want %q", trustReport.WorkflowClass, "rag_answer")
	}
	if !trustReport.Pass {
		t.Fatalf("expected pass, got %+v", trustReport)
	}
	if trustReport.CASGrade == "" {
		t.Fatalf("expected CAS grade, got %+v", trustReport)
	}
	if trustReport.Chain.RecordCount != len(b.Records) {
		t.Fatalf("unexpected record count: got %d want %d", trustReport.Chain.RecordCount, len(b.Records))
	}
	if !trustReport.Chain.Valid {
		t.Fatalf("expected valid chain, got %+v", trustReport.Chain)
	}
	if len(trustReport.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(trustReport.Sections))
	}

	modelInvocation := trustReportSectionByTitle(t, trustReport.Sections, "Model invocation")
	if !modelInvocation.Pass {
		t.Fatalf("expected model invocation section to pass, got %+v", modelInvocation)
	}
	if modelInvocation.Fields["model_provider"] != "provider" {
		t.Fatalf("unexpected model provider field: %+v", modelInvocation.Fields)
	}
	if modelInvocation.Fields["model_id"] != "model-1" {
		t.Fatalf("unexpected model ID field: %+v", modelInvocation.Fields)
	}
	if !containsTrustReportNote(modelInvocation.Notes, "model_parameters_digest present") {
		t.Fatalf("expected model digest note, got %+v", modelInvocation.Notes)
	}

	response := trustReportSectionByTitle(t, trustReport.Sections, "Response")
	if !containsTrustReportNote(response.Notes, "ai.response.sent present, request_id bound") {
		t.Fatalf("expected bound response note, got %+v", response.Notes)
	}
}

func TestTrustReportFromVerify_BackgroundAutomation(t *testing.T) {
	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIJobScheduled, map[string]any{
		"job_id":               "job-1",
		"job_type":             "nightly_sync",
		"trigger_source":       "cron",
		"scheduled_by_id_hash": "scheduler-hash",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobStarted, map[string]any{
		"job_id":         "job-1",
		"worker_id_hash": "worker-hash",
		"started_at":     "2026-03-27T12:01:00Z",
	}, "2026-03-27T12:01:00Z")
	appendVerifyRecord(t, b, event.TypeAIJobCompleted, map[string]any{
		"job_id":            "job-1",
		"outcome":           "success",
		"completion_reason": "completed",
	}, "2026-03-27T12:02:00Z")

	report := Verify(b, "bundle.atb", "")
	trustReport := TrustReportFromVerify(report, b)

	if trustReport.ProfileID != profileIDBackgroundAutomation {
		t.Fatalf("unexpected profile ID: got %q want %q", trustReport.ProfileID, profileIDBackgroundAutomation)
	}
	if trustReport.CASGrade == "" {
		t.Fatalf("expected CAS grade, got %+v", trustReport)
	}
	if len(trustReport.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(trustReport.Sections))
	}

	jobScheduling := trustReportSectionByTitle(t, trustReport.Sections, "Job scheduling")
	if !jobScheduling.Pass {
		t.Fatalf("expected job scheduling section to pass, got %+v", jobScheduling)
	}
	if jobScheduling.Fields["job_id"] != "job-1" || jobScheduling.Fields["job_type"] != "nightly_sync" {
		t.Fatalf("unexpected job scheduling fields: %+v", jobScheduling.Fields)
	}

	jobExecution := trustReportSectionByTitle(t, trustReport.Sections, "Job execution")
	if !jobExecution.Pass {
		t.Fatalf("expected job execution section to pass, got %+v", jobExecution)
	}
	if jobExecution.Fields["outcome"] != "success" {
		t.Fatalf("unexpected job execution fields: %+v", jobExecution.Fields)
	}
	if !containsTrustReportNote(jobExecution.Notes, "job_id consistent across scheduled/started/completed") {
		t.Fatalf("expected job consistency note, got %+v", jobExecution.Notes)
	}
}

func trustReportSectionByTitle(t testing.TB, sections []TrustSection, title string) TrustSection {
	t.Helper()
	for _, section := range sections {
		if section.Title == title {
			return section
		}
	}
	t.Fatalf("section %q not found in %+v", title, sections)
	return TrustSection{}
}

func containsTrustReportNote(notes []string, want string) bool {
	for _, note := range notes {
		if note == want {
			return true
		}
	}
	return false
}

func TestFirstEventField(t *testing.T) {
	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id": "req-1",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id": "req-2",
	}, "2026-03-27T12:01:00Z")

	if got := firstEventField(b, event.TypeAIRequestReceived, "request_id"); got != "req-1" {
		t.Fatalf("unexpected field value: got %q want %q", got, "req-1")
	}
	if got := firstEventField(b, event.TypeAIPolicyDecision, "policy_id"); got != "" {
		t.Fatalf("expected missing field to return empty string, got %q", got)
	}
}

func TestTrustReportPolicyDocSignatureValid_NilWhenNoPolicyDocHash(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)
	report := Verify(b, "bundle.atb", profileIDPrivilegedToolAction)
	trustReport := TrustReportFromVerify(report, b)

	if trustReport.PolicyDocSignatureValid != nil {
		t.Errorf("expected nil PolicyDocSignatureValid when no policy_doc_hash present, got %v", *trustReport.PolicyDocSignatureValid)
	}
}

func TestTrustReportPolicyDocSignatureValid_TrueWhenVerified(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	docContents := []byte("policy document v1")
	sum := sha256.Sum256(docContents)
	docHash := hex.EncodeToString(sum[:])

	fields := map[string]any{
		"policy_id":                          "pol-1",
		"policy_version":                     "2026-04",
		"decision":                           "allow",
		"decision_reason_codes":              []any{"ticket_present"},
		"subject_id_hash":                    "subject-hash",
		"action_id":                          "act-1",
		event.FieldPolicyDocHash:             docHash,
	}
	sig, err := signpkg.SignPolicyDoc(fields, docHash, privateKey)
	if err != nil {
		t.Fatalf("SignPolicyDoc: %v", err)
	}
	fields[event.FieldPolicyDocSignature] = sig
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)

	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIPolicyDecision, fields, "2026-03-27T12:00:00Z")

	report := Verify(b, "bundle.atb", "")
	trustReport := TrustReportFromVerify(report, b)

	if trustReport.PolicyDocSignatureValid == nil {
		t.Fatal("expected non-nil PolicyDocSignatureValid")
	}
	if !*trustReport.PolicyDocSignatureValid {
		t.Error("expected PolicyDocSignatureValid=true")
	}
}

func TestTrustReportPolicyDocSignatureValid_FalseWhenDocHashPresentButSignatureAbsent(t *testing.T) {
	sum := sha256.Sum256([]byte("doc"))
	fields := map[string]any{
		"policy_id":            "pol-1",
		"policy_version":       "2026-04",
		"decision":             "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":      "subject-hash",
		"action_id":            "act-1",
		event.FieldPolicyDocHash: hex.EncodeToString(sum[:]),
	}

	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, event.TypeAIPolicyDecision, fields, "2026-03-27T12:00:00Z")

	report := Verify(b, "bundle.atb", "")
	trustReport := TrustReportFromVerify(report, b)

	if trustReport.PolicyDocSignatureValid == nil {
		t.Fatal("expected non-nil PolicyDocSignatureValid")
	}
	if *trustReport.PolicyDocSignatureValid {
		t.Error("expected PolicyDocSignatureValid=false when signature absent")
	}
}

func TestFirstEventField_NonObjectDataReturnsEmpty(t *testing.T) {
	b := newVerifyTestBundle(t)
	appendVerifyRecord(t, b, bundle.AnchorEventType, `{"bundle_hash":"bundle-hash"}`, "2026-03-27T12:00:00Z")

	if got := firstEventField(b, bundle.AnchorEventType, "bundle_hash"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
