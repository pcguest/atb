package verify

import (
	"encoding/json"
	"math"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
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
