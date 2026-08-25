// SPDX-License-Identifier: MIT
package compliancepack_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/compliancepack"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/retentionaudit"
)

func TestBuildIncludesProfileAndIncidentEvidence(t *testing.T) {
	path := writePolicyBundle(t)
	pack, err := compliancepack.Build(
		context.Background(),
		path,
		"atb.profile.policy_decision",
		compliancepack.RegimeEUAIAct,
		"9.9.9",
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	files := map[string][]byte{}
	for _, file := range pack.Files {
		files[file.Name] = file.Content
	}
	for _, name := range []string{
		"bundle.atb",
		"reports/verify.report.json",
		"reports/trust-report.json",
		"reports/cas.json",
		"reports/obligations.json",
		"incidents/index.json",
		"incidents/sess-review.json",
		"retention/operations.atb",
		"retention/events.json",
		"docs/compliance/article-12-mapping.md",
		"MANIFEST.json",
		"SHA256SUMS",
	} {
		if _, ok := files[name]; !ok {
			t.Errorf("pack missing %q", name)
		}
	}

	var manifest compliancepack.Manifest
	if err := json.Unmarshal(files["MANIFEST.json"], &manifest); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if manifest.ProfileID != "atb.profile.policy_decision" ||
		!manifest.IntegrityPass || !manifest.ProfilePass {
		t.Fatalf("manifest = %+v", manifest)
	}
	for _, entry := range manifest.Files {
		content, ok := files[entry.Name]
		if !ok {
			t.Errorf("manifest references missing %q", entry.Name)
			continue
		}
		sum := sha256.Sum256(content)
		if entry.SHA256 != hex.EncodeToString(sum[:]) {
			t.Errorf("digest mismatch for %q", entry.Name)
		}
	}
}

func writePolicyBundle(t *testing.T) string {
	t.Helper()
	b, err := bundle.New()
	if err != nil {
		t.Fatal(err)
	}
	add := func(eventType string, data map[string]any) {
		t.Helper()
		if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{
			Timestamp: "2026-06-15T00:00:00Z",
		}); err != nil {
			t.Fatalf("append %s: %v", eventType, err)
		}
	}
	add("ai.request.received", map[string]any{
		"session_id": "sess-review", "request_id": "req-1",
		"actor_id_hash": "sha256:actor", "purpose_tag": "policy_decision",
	})
	add("ai.action.precommit", map[string]any{
		"session_id": "sess-review", "action_id": "act-1",
		"action_type": "deploy", "action_parameters_digest": "sha256:params",
		"target_resource_id": "prod", "intended_effect": "deploy",
	})
	add("ai.policy.decision", map[string]any{
		"session_id": "sess-review", "action_id": "act-1",
		"policy_id": "deploy-policy", "policy_version": "v1",
		"decision": "deny", "decision_reason_codes": []any{"manual_review"},
		"subject_id_hash": "sha256:actor",
		"identity_evidence": map[string]any{
			"identity_provider": "https://idp.example",
			"subject":           "reviewer-1", "assertion_type": "jwt",
			"assertion_digest": "sha256:assertion",
		},
	})
	add("atb.session.close", map[string]any{
		"session_id": "sess-review", "actor_id": "reviewer-1",
	})
	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	headHash := b.Records[len(b.Records)-1].Hash
	if err := retentionaudit.Append(
		retentionaudit.PathForBundle(path),
		event.TypeDataRetentionEnforced,
		map[string]any{
			"operation":              "s3_object_lock_request",
			"enforcement_system":     "s3",
			"outcome":                "request_accepted",
			"evidence_level":         "remote_api_acceptance",
			"independently_verified": false,
			"bundle_hash":            headHash,
		},
		time.Date(2026, 6, 15, 0, 1, 0, 0, time.UTC),
	); err != nil {
		t.Fatal(err)
	}
	return path
}
