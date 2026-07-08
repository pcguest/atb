// SPDX-License-Identifier: MIT
package emit

import (
	"errors"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

func TestEmitterOptionalFieldsAndIdentityEvidence(t *testing.T) {
	var captured []event.Event
	emitter, err := NewEmitter(" session-default ", func(evt event.Event) (string, error) {
		captured = append(captured, evt)
		return "hash", nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := emitter.ToolCall(ToolCallOptions{
		ToolName: " shell ", ActorID: " operator ", SessionID: " override ",
		InputDigest: "in", OutputDigest: "out",
	}); err != nil {
		t.Fatal(err)
	}
	if err := emitter.DataExport(DataExportOptions{
		ExportTarget: " archive ", ActorID: "operator", RecordCount: 3, Classification: "restricted",
	}); err != nil {
		t.Fatal(err)
	}
	evidence := &IdentityEvidence{
		IdentityProvider: " oidc ", Subject: " user-1 ", AuthContext: " mfa ",
		AssertionType: " jwt ", AssertionDigest: " sha256:assertion ", RawEvidenceDigest: " sha256:raw ",
	}
	if err := emitter.HumanOverride(HumanOverrideOptions{
		OverrideReason: " emergency ", ActorID: "operator", OverriddenActionID: "action-1", IdentityEvidence: evidence,
	}); err != nil {
		t.Fatal(err)
	}
	if err := emitter.HumanApproval(HumanApprovalOptions{
		ApprovedActionID: " action-2 ", ActorID: "operator", ApproverID: "reviewer", Note: "approved", IdentityEvidence: evidence,
	}); err != nil {
		t.Fatal(err)
	}
	if len(captured) != 4 {
		t.Fatalf("captured=%d", len(captured))
	}
	toolData := captured[0].Data.(map[string]any)
	if toolData["session_id"] != "override" || toolData["tool_input_digest"] != "in" {
		t.Fatalf("tool data=%v", toolData)
	}
	exportData := captured[1].Data.(map[string]any)
	if exportData["record_count"] != 3 || exportData["classification"] != "restricted" {
		t.Fatalf("export data=%v", exportData)
	}
	overrideData := captured[2].Data.(map[string]any)
	identityData := overrideData["identity_evidence"].(map[string]any)
	if identityData["auth_context"] != "mfa" || identityData["raw_evidence_digest"] != "sha256:raw" {
		t.Fatalf("identity data=%v", identityData)
	}
}

func TestEmitterValidationAndAppendErrors(t *testing.T) {
	if _, err := NewEmitter("session", nil); err == nil {
		t.Fatal("nil append function accepted")
	}
	appendErr := errors.New("append failed")
	emitter, err := NewEmitter("session", func(event.Event) (string, error) {
		return "", appendErr
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := emitter.ToolCall(ToolCallOptions{ToolName: "tool"}); !errors.Is(err, appendErr) {
		t.Fatalf("append error=%v", err)
	}
	if err := emitter.DataExport(DataExportOptions{}); err == nil {
		t.Fatal("empty export target accepted")
	}
	if err := emitter.HumanOverride(HumanOverrideOptions{}); err == nil {
		t.Fatal("empty override reason accepted")
	}

	invalidEvidence := []*IdentityEvidence{
		{},
		{IdentityProvider: "idp"},
		{IdentityProvider: "idp", Subject: "subject"},
		{IdentityProvider: "idp", Subject: "subject", AssertionType: "jwt"},
	}
	for i, evidence := range invalidEvidence {
		err := emitter.HumanApproval(HumanApprovalOptions{
			ApprovedActionID: "action", IdentityEvidence: evidence,
		})
		if err == nil {
			t.Fatalf("invalid evidence %d accepted", i)
		}
	}
}

func TestIdentityEvidenceDataMinimalAndNil(t *testing.T) {
	if got, err := identityEvidenceData(nil); err != nil || got != nil {
		t.Fatalf("nil evidence=%v err=%v", got, err)
	}
	got, err := identityEvidenceData(&IdentityEvidence{
		IdentityProvider: "idp", Subject: "subject",
		AssertionType: "jwt", AssertionDigest: "sha256:value",
	})
	if err != nil || len(got) != 4 {
		t.Fatalf("minimal evidence=%v err=%v", got, err)
	}
}
