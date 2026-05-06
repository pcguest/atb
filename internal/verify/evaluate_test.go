// SPDX-License-Identifier: MIT
package verify

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestEvaluateBundleMatchesLegacyVerify(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	tests := []struct {
		name  string
		build func(*testing.T) *bundle.Bundle
	}{
		{
			name: "healthy bundle",
			build: func(t *testing.T) *bundle.Bundle {
				return newPrivilegedToolActionBundle(t)
			},
		},
		{
			name: "broken chain",
			build: func(t *testing.T) *bundle.Bundle {
				b := newPrivilegedToolActionBundle(t)
				b.Records[1].Event.Type = "ai.action.precommit.tampered"
				return b
			},
		},
		{
			name: "obligation failure",
			build: func(t *testing.T) *bundle.Bundle {
				b := newVerifyTestBundle(t)
				appendVerifyRecord(t, b, "ai.request.received", map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "approve-change",
				}, "2026-03-27T12:00:00Z")
				appendVerifyRecord(t, b, "ai.action.precommit", map[string]any{
					"action_id":                "act-1",
					"action_type":              "deploy_change",
					"action_parameters_digest": "params-digest",
					"target_resource_id":       "svc-1",
					"intended_effect":          "deploy build 42",
				}, "2026-03-27T12:01:00Z")
				appendVerifyRecord(t, b, "ai.policy.decision", map[string]any{
					"policy_id":             "pol-1",
					"policy_version":        "2026-03",
					"decision":              "allow",
					"decision_reason_codes": []any{"ticket_present"},
					"subject_id_hash":       "subject-hash",
					"action_id":             "act-1",
				}, "2026-03-27T12:02:00Z")
				appendVerifyRecord(t, b, "ai.action.executed", map[string]any{
					"action_id":           "act-1",
					"execution_outcome":   "success",
					"tool_receipt_digest": "tool-digest",
				}, "2026-03-27T12:05:00Z")
				appendVerifyRecord(t, b, "ai.action.committed", map[string]any{
					"action_id":           "act-1",
					"commit_outcome":      "success",
					"sink_receipt_digest": "sink-digest",
				}, "2026-03-27T12:06:00Z")
				return b
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.build(t)
			bundlePath := writeEvaluateBundleFixture(t, b)

			want := VerifyWithProfile(b, bundlePath, profile)

			fromPath, err := EvaluateBundle(EvaluateConfig{
				BundlePath: bundlePath,
				Profiles:   []Profile{profile},
			})
			if err != nil {
				t.Fatalf("EvaluateBundle(path): %v", err)
			}

			fromRecords, err := EvaluateBundle(EvaluateConfig{
				BundlePath: bundlePath,
				Records:    b.Records,
				Profiles:   []Profile{profile},
			})
			if err != nil {
				t.Fatalf("EvaluateBundle(records): %v", err)
			}

			wantJSON := mustMarshalReportJSON(t, want)
			if got := mustMarshalReportJSON(t, *fromPath); string(got) != string(wantJSON) {
				t.Fatalf("path report JSON changed:\n got: %s\nwant: %s", got, wantJSON)
			}
			if got := mustMarshalReportJSON(t, *fromRecords); string(got) != string(wantJSON) {
				t.Fatalf("record report JSON changed:\n got: %s\nwant: %s", got, wantJSON)
			}
		})
	}
}

func TestEvaluateBundleFailureIDs(t *testing.T) {
	profile := ProfileByID(profileIDDataExport)
	if profile == nil {
		t.Fatal("expected data export profile")
	}

	t.Run("missing_required_event", func(t *testing.T) {
		report := VerifyWithProfile(newVerifyTestBundle(t), "bundle.atb", profile)
		assertReportFailureID(
			t,
			report,
			"missing_event",
			event.TypeAIRequestReceived,
			"required:ai.request.received",
		)
	})

	t.Run("missing_required_field", func(t *testing.T) {
		b := newVerifyTestBundle(t)
		appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
			"actor_id_hash": "actor-hash",
			"purpose_tag":   "data_export",
		}, "2026-03-27T12:00:00Z")

		report := VerifyWithProfile(b, "bundle.atb", profile)
		assertReportFailureID(
			t,
			report,
			"missing_field",
			event.TypeAIRequestReceived,
			"required:ai.request.received:field:request_id",
		)
	})

	t.Run("relation_violation", func(t *testing.T) {
		b := newVerifyTestBundle(t)
		appendDataExportCoreRecords(t, b, "act-1", "act-2")
		appendVerifyRecord(t, b, event.TypeAIHumanApproval, map[string]any{
			"approval_id":          "appr-1",
			"approver_id_hash":     "approver-hash",
			"approval_outcome":     "approved",
			"justification_digest": "just-digest",
			"action_id":            "act-2",
		}, "2026-03-27T12:04:00Z")

		report := VerifyWithProfile(b, "bundle.atb", profile)
		assertReportFailureID(
			t,
			report,
			"relation_violation",
			"execution_after_authorization",
			"relation:execution_after_authorization",
		)
	})

	t.Run("required_when", func(t *testing.T) {
		b := newVerifyTestBundle(t)
		appendDataExportCoreRecords(t, b, "act-1", "act-1")

		report := VerifyWithProfile(b, "bundle.atb", profile)
		got := reportFailureID(report, "missing_event", "ai.human.approval required when data exports execute")
		if !strings.HasPrefix(got, "required_when:") {
			t.Fatalf("required_when failure ID = %q, want prefix %q", got, "required_when:")
		}
	})
}

func TestEvaluateReportFromVerifyCopiesFailureID(t *testing.T) {
	report := Report{
		Integrity: IntegrityResult{ChainValid: true},
		Profiles: []ProfileResult{
			{
				ProfileID: "atb.profile.test",
				Pass:      false,
				CriticalFailures: []CriticalFailure{
					{
						ID:     "required:ai.request.received",
						Kind:   "missing_event",
						Detail: "ai.request.received missing",
					},
				},
			},
		},
	}

	verifierReport := ReportFromVerify(report)
	if got := verifierReport.Failures[0].ID; got != "required:ai.request.received" {
		t.Fatalf("ReportFromVerify failure ID = %q, want %q", got, "required:ai.request.received")
	}
}

func TestEvaluateBundleRequiresProfileSelection(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)

	_, err := EvaluateBundle(EvaluateConfig{
		BundlePath: "bundle.atb",
		Records:    b.Records,
	})
	if err == nil {
		t.Fatal("expected error when no profile selection is provided")
	}
	if err.Error() != "verify: no profiles supplied" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateBundleCorroborationNilPolicy(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	b := newPrivilegedToolActionBundle(t)
	bundlePath := writeEvaluateBundleFixture(t, b)

	// Without any option (nil policy) the bonus must be 0 and effective_score == overall.
	report, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	})
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	if report.CAS == nil {
		t.Fatal("expected non-nil CAS")
	}
	if report.CAS.CorroborationBonus != 0.0 {
		t.Errorf("nil policy: corroboration_bonus = %f, want 0.0", report.CAS.CorroborationBonus)
	}
	if report.CAS.EffectiveScore != report.CAS.Overall {
		t.Errorf("nil policy: effective_score = %f, want Overall = %f", report.CAS.EffectiveScore, report.CAS.Overall)
	}
	if report.CAS.Grade != gradeFromScore(report.CAS.Overall) {
		t.Errorf("nil policy: grade = %q, want %q", report.CAS.Grade, gradeFromScore(report.CAS.Overall))
	}
}

func TestEvaluateBundleCorroborationSignatureBonus(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	b := newPrivilegedToolActionBundle(t)
	bundlePath := writeEvaluateBundleFixture(t, b)
	addBundleSignatureRecord(t, b, bundlePath, privateKey)
	bundlePath = writeEvaluateBundleFixture(t, b)

	policy := DefaultCorroborationPolicy()
	report, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	}, WithCorroborationPolicy(policy))
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	if report.CAS == nil {
		t.Fatal("expected non-nil CAS")
	}
	if report.CAS.CorroborationBonus != policy.SignatureBonus {
		t.Errorf("signature bonus = %f, want SignatureBonus %f", report.CAS.CorroborationBonus, policy.SignatureBonus)
	}
	want := math.Min(1.0, report.CAS.Overall+policy.SignatureBonus)
	if math.Abs(report.CAS.EffectiveScore-want) > 1e-9 {
		t.Errorf("effective_score = %f, want %f", report.CAS.EffectiveScore, want)
	}
}

func TestEvaluateBundleCorroborationSignatureAndSnapshotBonus(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	b := newPrivilegedToolActionBundle(t)
	appendSnapshotRecord(t, b)
	bundlePath := writeEvaluateBundleFixture(t, b)
	addBundleSignatureRecord(t, b, bundlePath, privateKey)
	bundlePath = writeEvaluateBundleFixture(t, b)

	policy := DefaultCorroborationPolicy()
	report, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	}, WithCorroborationPolicy(policy))
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	if report.CAS == nil {
		t.Fatal("expected non-nil CAS")
	}

	wantBonus := policy.SignatureBonus + policy.SnapshotBonus
	if wantBonus > policy.MaxBonus {
		wantBonus = policy.MaxBonus
	}
	if math.Abs(report.CAS.CorroborationBonus-wantBonus) > 1e-9 {
		t.Errorf("sig+snapshot bonus = %f, want %f", report.CAS.CorroborationBonus, wantBonus)
	}
}

func TestEvaluateBundleCorroborationAllThreeCappedAtMaxBonus(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Use a custom policy where all three bonuses sum to exactly MaxBonus + ε,
	// so the cap is meaningfully exercised.
	policy := &CorroborationPolicy{
		AnchorBonus:    0.06,
		SignatureBonus: 0.04,
		SnapshotBonus:  0.03,
		MaxBonus:       0.10,
	}

	b := newPrivilegedToolActionBundle(t)
	appendSnapshotRecord(t, b)
	bundlePath := writeEvaluateBundleFixture(t, b)
	addBundleSignatureRecord(t, b, bundlePath, privateKey)
	bundlePath = writeEvaluateBundleFixture(t, b)

	report, err := EvaluateBundle(EvaluateConfig{
		BundlePath: bundlePath,
		Profiles:   []Profile{profile},
	}, WithCorroborationPolicy(policy))
	if err != nil {
		t.Fatalf("EvaluateBundle: %v", err)
	}
	if report.CAS == nil {
		t.Fatal("expected non-nil CAS")
	}

	// signature + snapshot = 0.07, which is less than MaxBonus of 0.10
	// The cap matters: if anchor were verified too, 0.06+0.04+0.03 = 0.13 → capped at 0.10
	// Without anchor, sig+snap = 0.07, not capped. Test validates cap doesn't exceed MaxBonus.
	if report.CAS.CorroborationBonus > policy.MaxBonus {
		t.Errorf("bonus = %f exceeds MaxBonus %f", report.CAS.CorroborationBonus, policy.MaxBonus)
	}
}

func addBundleSignatureRecord(t testing.TB, b *bundle.Bundle, bundlePath string, privateKey ed25519.PrivateKey) {
	t.Helper()

	pub := privateKey.Public().(ed25519.PublicKey)

	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle for signing: %v", err)
	}
	digest := sha256.Sum256(raw)
	sig := ed25519.Sign(privateKey, digest[:])

	if err := b.AppendWithOptions(event.TypeBundleSignature, map[string]any{
		"algorithm":   "ed25519",
		"pubkey":      base64.StdEncoding.EncodeToString(pub),
		"signature":   base64.StdEncoding.EncodeToString(sig),
		"bundle_hash": hex.EncodeToString(digest[:]),
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:10:00Z"}); err != nil {
		t.Fatalf("append bundle signature: %v", err)
	}
}

func appendDataExportCoreRecords(t testing.TB, b *bundle.Bundle, policyActionID string, executedActionID string) {
	t.Helper()

	appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-data-export",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "data_export",
	}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"export_allowed"},
		"subject_id_hash":       "subject-hash",
		"action_id":             policyActionID,
	}, "2026-03-27T12:01:00Z")
	appendVerifyRecord(t, b, event.TypeDataExportPrecommit, map[string]any{
		"action_id":                policyActionID,
		"action_type":              "export_data",
		"action_parameters_digest": "params-digest",
		"target_resource_id":       "dataset-1",
		"intended_effect":          "export approved dataset",
	}, "2026-03-27T12:02:00Z")
	appendVerifyRecord(t, b, event.TypeDataExportExecuted, map[string]any{
		"action_id":           executedActionID,
		"execution_outcome":   "success",
		"tool_receipt_digest": "tool-digest",
	}, "2026-03-27T12:03:00Z")
}

func appendSnapshotRecord(t testing.TB, b *bundle.Bundle) {
	t.Helper()

	bundleHash, err := SnapshotBundleHash(b.Records)
	if err != nil {
		t.Fatalf("compute snapshot hash: %v", err)
	}
	if err := b.AppendWithOptions(event.TypeSnapshot, map[string]any{
		"name":         "test-snapshot",
		"bundle_hash":  bundleHash,
		"record_count": len(b.Records),
		"snapshot_at":  "2026-03-27T12:09:00Z",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:09:00Z"}); err != nil {
		t.Fatalf("append snapshot: %v", err)
	}
}

func TestEvaluateBundleErrBundleNotFound(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	_, err := EvaluateBundle(EvaluateConfig{
		BundlePath: filepath.Join(t.TempDir(), "nonexistent.atb"),
		Profiles:   []Profile{profile},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent bundle path")
	}
	if !errors.Is(err, ErrBundleNotFound) {
		t.Fatalf("expected ErrBundleNotFound, got: %v", err)
	}
}

func TestEvaluateBundleErrChainInvalid(t *testing.T) {
	profile := ProfileByID(profileIDPrivilegedToolAction)
	if profile == nil {
		t.Fatal("expected built-in profile")
	}

	b := newPrivilegedToolActionBundle(t)
	b.Records[1].Event.Type = "ai.action.precommit.tampered"
	bundlePath := writeEvaluateBundleFixture(t, b)

	_, err := EvaluateBundle(EvaluateConfig{
		BundlePath:        bundlePath,
		Profiles:          []Profile{profile},
		RequireValidChain: true,
	})
	if err == nil {
		t.Fatal("expected error for invalid chain with RequireValidChain=true")
	}
	if !errors.Is(err, ErrChainInvalid) {
		t.Fatalf("expected ErrChainInvalid, got: %v", err)
	}
}

func TestEvaluateBundleErrProfileUnknown(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)

	_, err := EvaluateBundle(EvaluateConfig{
		BundlePath: "bundle.atb",
		Records:    b.Records,
		Profiles:   []Profile{nil},
	})
	if err == nil {
		t.Fatal("expected error when all supplied profiles are nil")
	}
	if !errors.Is(err, ErrProfileUnknown) {
		t.Fatalf("expected ErrProfileUnknown, got: %v", err)
	}
}

func writeEvaluateBundleFixture(t testing.TB, b *bundle.Bundle) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle fixture: %v", err)
	}
	return path
}

func mustMarshalReportJSON(t testing.TB, report Report) []byte {
	t.Helper()

	data, err := json.Marshal(normaliseReportForJSONComparison(report))
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return data
}

func normaliseReportForJSONComparison(report Report) Report {
	if report.CAS == nil {
		return report
	}

	cloned := report
	cas := *report.CAS
	cas.Overall = roundComparisonFloat(cas.Overall)
	cas.CorroborationBonus = roundComparisonFloat(cas.CorroborationBonus)
	cas.EffectiveScore = roundComparisonFloat(cas.EffectiveScore)
	cas.SubScores = cloneAndRoundFloatMap(cas.SubScores)
	cas.WeightVector = cloneAndRoundFloatMap(cas.WeightVector)
	cloned.CAS = &cas
	return cloned
}

func cloneAndRoundFloatMap(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return values
	}

	cloned := make(map[string]float64, len(values))
	for key, value := range values {
		cloned[key] = roundComparisonFloat(value)
	}
	return cloned
}

func roundComparisonFloat(value float64) float64 {
	const scale = 1e12
	return math.Round(value*scale) / scale
}

func assertReportFailureID(t testing.TB, report Report, kind string, containsText string, want string) {
	t.Helper()

	if got := reportFailureID(report, kind, containsText); got != want {
		t.Fatalf("failure ID = %q, want %q", got, want)
	}
}

func reportFailureID(report Report, kind string, containsText string) string {
	if len(report.Profiles) == 0 {
		return ""
	}
	for _, failure := range report.Profiles[0].CriticalFailures {
		if failure.Kind == kind && strings.Contains(failure.Detail, containsText) {
			return failure.ID
		}
	}
	return ""
}
