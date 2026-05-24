// SPDX-License-Identifier: MIT
package verify_test

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/verify"
)

func TestStrictSourceSignatures_AbsentPolicySignatureFails(t *testing.T) {
	t.Parallel()

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	if err := b.Append(event.TypeAIRequestReceived, map[string]any{"request_id": "r1", "actor_id_hash": "a1", "purpose_tag": "policy_decision"}); err != nil {
		t.Fatalf("append request: %v", err)
	}
	if err := b.Append(event.TypeAIPolicyDecision, map[string]any{
		"policy_id": "p1", "policy_version": "1", "decision": "allow",
		"decision_reason_codes": []string{"ok"}, "subject_id_hash": "a1", "action_id": "act1",
	}); err != nil {
		t.Fatalf("append policy: %v", err)
	}

	report, err := verify.EvaluateBundle(verify.EvaluateConfig{
		Records:                b.Records,
		Profiles:               []verify.Profile{verify.ProfileByID("atb.profile.policy_decision")},
		StrictSourceSignatures: true,
	})
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	if len(report.Profiles) == 0 || report.Profiles[0].Pass {
		t.Fatalf("expected profile fail under strict signatures, got %+v", report.Profiles)
	}
}
