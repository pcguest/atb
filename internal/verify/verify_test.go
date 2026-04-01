package verify

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"math"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/hash"
	signpkg "github.com/pcguest/atb/internal/sign"
)

func TestVerify_IntegrityFail(t *testing.T) {
	b := bundle.New()
	appendVerifyRecord(t, b, "dev.session", map[string]any{"event_id": "evt-1"}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, "dev.session", map[string]any{"event_id": "evt-2"}, "2026-03-27T12:01:00Z")
	b.Records[1].Event.Type = "dev.session.tampered"

	report := Verify(b, "bundle.atb", "")
	if report.Integrity.ChainValid {
		t.Fatalf("expected integrity failure")
	}
	if report.CAS != nil {
		t.Fatalf("expected nil CAS on integrity failure")
	}
	if report.ResidualRisk.Level != "High" {
		t.Fatalf("unexpected residual risk level: got %q want %q", report.ResidualRisk.Level, "High")
	}
}

func TestVerify_NoProfile(t *testing.T) {
	b := bundle.New()
	appendVerifyRecord(t, b, "dev.session", map[string]any{"event_id": "evt-1"}, "2026-03-27T12:00:00Z")
	appendVerifyRecord(t, b, "dev.session", map[string]any{"event_id": "evt-2"}, "2026-03-27T12:01:00Z")

	report := Verify(b, "bundle.atb", "")
	if !report.Integrity.ChainValid {
		t.Fatalf("expected integrity pass")
	}
	if len(report.Profiles) != 0 {
		t.Fatalf("expected no matched profiles, got %d", len(report.Profiles))
	}
	if report.CAS == nil {
		t.Fatalf("expected fallback CAS when no profile matches")
	}
	if _, ok := report.CAS.SubScores["SC"]; !ok {
		t.Fatalf("expected fallback CAS to include SC")
	}
}

func TestVerify_ProfileAutoDetect(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)

	report := Verify(b, "bundle.atb", "")
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one matched profile, got %d", len(report.Profiles))
	}
	if report.Profiles[0].ProfileID != profileIDPrivilegedToolAction {
		t.Fatalf("unexpected profile ID: got %q want %q", report.Profiles[0].ProfileID, profileIDPrivilegedToolAction)
	}
	if !report.Profiles[0].Pass {
		t.Fatalf("expected profile pass, got failures %+v", report.Profiles[0].CriticalFailures)
	}
}

func TestVerify_PolicyDecision_Signed(t *testing.T) {
	b := newSignedPrivilegedToolActionBundle(t)

	report := Verify(b, "bundle.atb", profileIDPrivilegedToolAction)
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one profile result, got %d", len(report.Profiles))
	}
	if !hasInformationalNote(report.Profiles[0].InformationalNotes, "ai.policy.decision: signature verified") {
		t.Fatalf("expected signature verified note, got %v", report.Profiles[0].InformationalNotes)
	}
	if hasRequiredWarning(report.Profiles[0].RequiredWarnings, "ai.policy.decision: policy_signature absent") {
		t.Fatalf("did not expect unsigned warning, got %v", report.Profiles[0].RequiredWarnings)
	}
}

func TestVerify_PolicyDecision_Unsigned(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)

	report := Verify(b, "bundle.atb", profileIDPrivilegedToolAction)
	if len(report.Profiles) != 1 {
		t.Fatalf("expected one profile result, got %d", len(report.Profiles))
	}
	if !hasRequiredWarning(report.Profiles[0].RequiredWarnings, "ai.policy.decision: policy_signature absent") {
		t.Fatalf("expected unsigned warning, got %v", report.Profiles[0].RequiredWarnings)
	}
}

func TestVerify_CAS_GradeThresholds(t *testing.T) {
	weights := map[string]float64{
		"EC": 0.125,
		"FC": 0.125,
		"RC": 0.125,
		"TC": 0.125,
		"SC": 0.125,
		"XC": 0.125,
		"AC": 0.125,
		"GC": 0.125,
	}

	tests := []struct {
		score float64
		grade string
	}{
		{score: 0.90, grade: "High"},
		{score: 0.70, grade: "Medium"},
		{score: 0.45, grade: "Low"},
		{score: 0.20, grade: "Insufficient"},
	}

	for _, tc := range tests {
		subScores := map[string]float64{
			"EC": tc.score,
			"FC": tc.score,
			"RC": tc.score,
			"TC": tc.score,
			"SC": tc.score,
			"XC": tc.score,
			"AC": tc.score,
			"GC": tc.score,
		}
		result := ComputeCAS(subScores, weights, true)
		if result.Grade != tc.grade {
			t.Fatalf("score %.2f produced grade %q want %q", tc.score, result.Grade, tc.grade)
		}
	}
}

func TestComputeSC(t *testing.T) {
	tolerance := 0.001

	tests := []struct {
		name      string
		build     func(testing.TB) *bundle.Bundle
		profileID string
		want      float64
		wantCAS   bool
	}{
		{
			name:      "anchor_and_manifest_and_request_privileged",
			profileID: profileIDPrivilegedToolAction,
			want:      1.0,
			wantCAS:   true,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				// With policy present this still reaches the clamp without a signature:
				// 0.40 + 0.25 + 0.20 + 0.15 = 1.00.
				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "approve-change",
				}, "2026-03-27T12:00:00Z")
				appendVerifyRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id":             "pol-1",
					"policy_version":        "2026-03",
					"decision":              "allow",
					"decision_reason_codes": []any{"ticket_present"},
					"subject_id_hash":       "subject-hash",
					"action_id":             "act-1",
				}, "2026-03-27T12:01:00Z")
				appendVerifyRecord(t, b, event.TypeBundleAnchor,
					`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:01:30Z"}`,
					"2026-03-27T12:01:30Z")
				return b
			},
		},
		{
			name:      "with_signature_record",
			profileID: profileIDPrivilegedToolAction,
			want:      math.Min(0.25+0.40+0.20+0.10, 1.0),
			wantCAS:   true,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "approve-change",
				}, "2026-03-27T12:00:00Z")
				appendVerifyRecord(t, b, event.TypeBundleAnchor,
					`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:01:30Z"}`,
					"2026-03-27T12:01:30Z")
				appendVerifyRecord(t, b, event.TypeBundleSignature, map[string]any{
					"algorithm":   "ed25519",
					"public_key":  "cHVibGljLWtleQ==",
					"signature":   "c2lnbmF0dXJl",
					"bundle_hash": "bundle-hash",
				}, "2026-03-27T12:01:45Z")
				return b
			},
		},
		{
			name:      "anchor_and_manifest_and_request_rag",
			profileID: profileIDRAGAnswer,
			want:      1.0,
			wantCAS:   true,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "answer-question",
				}, "2026-03-27T12:00:00Z")
				appendVerifyRecord(t, b, event.TypeAIRetrievalExecuted, map[string]any{
					"retrieval_query_hash":     "query-hash",
					"retrieval_corpus_id":      "corpus-1",
					"retrieval_corpus_version": "2026-03",
					"top_k":                    5,
					"result_set_digest":        "result-digest",
				}, "2026-03-27T12:01:00Z")
				appendVerifyRecord(t, b, event.TypeBundleAnchor,
					`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:01:30Z"}`,
					"2026-03-27T12:01:30Z")
				return b
			},
		},
		{
			name:      "no_anchor",
			profileID: profileIDPrivilegedToolAction,
			want:      0.45,
			wantCAS:   true,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "approve-change",
				}, "2026-03-27T12:00:00Z")
				return b
			},
		},
		{
			name:      "manifest_only",
			profileID: profileIDPrivilegedToolAction,
			want:      0.25,
			wantCAS:   true,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()
				return bundle.New()
			},
		},
		{
			name:      "empty_bundle",
			profileID: profileIDPrivilegedToolAction,
			want:      0.0,
			wantCAS:   true,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()
				return &bundle.Bundle{}
			},
		},
		{
			name:      "manifest_not_at_seq_0",
			profileID: profileIDPrivilegedToolAction,
			want:      0.60,
			wantCAS:   true,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()
				return newVerifyBundleFromEvents(t, []event.Event{
					{
						Sequence:  1,
						Type:      event.TypeBundleAnchor,
						HashAlgo:  "sha256",
						Data:      `{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:00:30Z"}`,
						Timestamp: "2026-03-27T12:00:30Z",
					},
					{
						Sequence:  2,
						Type:      event.TypeBundleManifest,
						HashAlgo:  "sha256",
						Data:      `{"version":"1","created_at":"2026-03-27T12:00:00Z","bundle_id":"bundle-1"}`,
						Timestamp: "2026-03-27T12:00:00Z",
					},
					{
						Sequence: 3,
						Type:     event.TypeAIRequestReceived,
						HashAlgo: "sha256",
						Data: map[string]any{
							"request_id":    "req-1",
							"actor_id_hash": "actor-hash",
							"purpose_tag":   "approve-change",
						},
						Timestamp: "2026-03-27T12:01:00Z",
					},
				})
			},
		},
		{
			name:      "unknown_profile",
			profileID: "unknown.profile.id",
			want:      0.0,
			wantCAS:   false,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "approve-change",
				}, "2026-03-27T12:00:00Z")
				appendVerifyRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id":             "pol-1",
					"policy_version":        "2026-03",
					"decision":              "allow",
					"decision_reason_codes": []any{"ticket_present"},
					"subject_id_hash":       "subject-hash",
					"action_id":             "act-1",
				}, "2026-03-27T12:01:00Z")
				appendVerifyRecord(t, b, event.TypeAIRetrievalExecuted, map[string]any{
					"retrieval_query_hash":     "query-hash",
					"retrieval_corpus_id":      "corpus-1",
					"retrieval_corpus_version": "2026-03",
					"top_k":                    5,
					"result_set_digest":        "result-digest",
				}, "2026-03-27T12:02:00Z")
				appendVerifyRecord(t, b, event.TypeBundleAnchor,
					`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:02:30Z"}`,
					"2026-03-27T12:02:30Z")
				return b
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.build(t)

			got := computeSC(b, tc.profileID)
			if diff := math.Abs(got - tc.want); diff > tolerance {
				t.Fatalf("computeSC() = %.3f, want %.3f (diff %.3f)", got, tc.want, diff)
			}

			report := Verify(b, "bundle.atb", tc.profileID)
			if !tc.wantCAS {
				if report.CAS != nil {
					t.Fatalf("expected nil CAS for profile %q, got %+v", tc.profileID, report.CAS)
				}
				return
			}

			if report.CAS == nil {
				t.Fatalf("expected CAS for profile %q", tc.profileID)
			}
			if diff := math.Abs(report.CAS.SubScores["SC"] - tc.want); diff > tolerance {
				t.Fatalf("report CAS SC = %.3f, want %.3f (diff %.3f)", report.CAS.SubScores["SC"], tc.want, diff)
			}
		})
	}
}

func TestCASSubScoresSC(t *testing.T) {
	tests := []struct {
		name      string
		build     func(testing.TB) *bundle.Bundle
		profileID string
		check     func(*testing.T, Report)
	}{
		{
			name:      "sc_written_to_sub_scores_map",
			profileID: profileIDPrivilegedToolAction,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeBundleSignature, map[string]any{
					"algorithm":   "ed25519",
					"public_key":  "cHVibGljLWtleQ==",
					"signature":   "c2lnbmF0dXJl",
					"bundle_hash": "bundle-hash",
				}, "2026-03-27T12:00:00Z")
				return b
			},
			check: func(t *testing.T, report Report) {
				t.Helper()
				if report.CAS == nil {
					t.Fatalf("expected CAS for explicit profile")
				}
				sc, ok := report.CAS.SubScores["SC"]
				if !ok {
					t.Fatalf("expected SC sub-score key to be present")
				}
				if sc < 0.25 {
					t.Fatalf("expected SC >= 0.25, got %.3f", sc)
				}
			},
		},
		{
			name:      "explicit_profile_manifest_only",
			profileID: profileIDPrivilegedToolAction,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeDevSession, map[string]any{
					"event_id": "evt-1",
				}, "2026-03-27T12:00:00Z")
				appendVerifyRecord(t, b, event.TypeBundleSignature, map[string]any{
					"algorithm":   "ed25519",
					"public_key":  "cHVibGljLWtleQ==",
					"signature":   "c2lnbmF0dXJl",
					"bundle_hash": "bundle-hash",
				}, "2026-03-27T12:00:01Z")
				return b
			},
			check: func(t *testing.T, report Report) {
				t.Helper()
				if report.CAS == nil {
					t.Fatalf("expected CAS for explicit profile")
				}
				sc, ok := report.CAS.SubScores["SC"]
				if !ok {
					t.Fatalf("expected SC sub-score to be present")
				}
				if sc < 0.25 {
					t.Fatalf("expected SC >= 0.25, got %.3f", sc)
				}
				if sc > 0.40 {
					t.Fatalf("expected SC <= 0.40, got %.3f", sc)
				}
			},
		},
		{
			name:      "explicit_profile_full_privileged",
			profileID: profileIDPrivilegedToolAction,
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeBundleAnchor,
					`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:00:30Z"}`,
					"2026-03-27T12:00:30Z")
				appendVerifyRecord(t, b, event.TypeAIRequestReceived, map[string]any{
					"request_id":    "req-1",
					"actor_id_hash": "actor-hash",
					"purpose_tag":   "approve-change",
				}, "2026-03-27T12:01:00Z")
				appendVerifyRecord(t, b, event.TypeAIPolicyDecision, map[string]any{
					"policy_id":             "pol-1",
					"policy_version":        "2026-03",
					"decision":              "allow",
					"decision_reason_codes": []any{"ticket_present"},
					"subject_id_hash":       "subject-hash",
					"action_id":             "act-1",
				}, "2026-03-27T12:01:30Z")
				appendVerifyRecord(t, b, event.TypeBundleSignature, map[string]any{
					"algorithm":   "ed25519",
					"public_key":  "cHVibGljLWtleQ==",
					"signature":   "c2lnbmF0dXJl",
					"bundle_hash": "bundle-hash",
				}, "2026-03-27T12:01:45Z")
				return b
			},
			check: func(t *testing.T, report Report) {
				t.Helper()
				if report.CAS == nil {
					t.Fatalf("expected CAS for explicit profile")
				}
				sc, ok := report.CAS.SubScores["SC"]
				if !ok {
					t.Fatalf("expected SC sub-score to be present")
				}
				if sc < 0.99 {
					t.Fatalf("expected SC >= 0.99, got %.3f", sc)
				}
			},
		},
		{
			name:      "no_profile_auto_detect_fallback",
			profileID: "",
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeDevSession, map[string]any{
					"event_id": "evt-1",
				}, "2026-03-27T12:00:00Z")
				return b
			},
			check: func(t *testing.T, report Report) {
				t.Helper()
				if report.CAS == nil {
					t.Fatalf("expected fallback CAS for unmatched auto-detect")
				}
				sc, ok := report.CAS.SubScores["SC"]
				if !ok {
					t.Fatalf("expected fallback SC sub-score to be present")
				}
				if sc < 0.0 {
					t.Fatalf("expected SC >= 0.0, got %.3f", sc)
				}
			},
		},
		{
			name:      "signed_bundle_sc_not_null",
			profileID: "",
			build: func(t testing.TB) *bundle.Bundle {
				t.Helper()

				b := bundle.New()
				appendVerifyRecord(t, b, event.TypeBundleSignature, map[string]any{
					"algorithm":   "ed25519",
					"public_key":  "cHVibGljLWtleQ==",
					"signature":   "c2lnbmF0dXJl",
					"bundle_hash": "bundle-hash",
				}, "2026-03-27T12:00:00Z")
				return b
			},
			check: func(t *testing.T, report Report) {
				t.Helper()
				if report.CAS == nil {
					t.Fatalf("expected fallback CAS for signed bundle")
				}
				sc, ok := report.CAS.SubScores["SC"]
				if !ok {
					t.Fatalf("expected fallback SC sub-score to be present")
				}
				if sc <= 0 {
					t.Fatalf("expected SC > 0, got %.3f", sc)
				}
				if report.CAS.Overall <= 0 {
					t.Fatalf("expected partial CAS overall > 0, got %.3f", report.CAS.Overall)
				}
				if report.CAS.Grade == "" {
					t.Fatalf("expected partial CAS grade to be populated")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := Verify(tc.build(t), "bundle.atb", tc.profileID)
			tc.check(t, report)
		})
	}
}

func appendVerifyRecord(t testing.TB, b *bundle.Bundle, eventType string, data interface{}, timestamp string) {
	t.Helper()
	if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
}

func newPrivilegedToolActionBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := bundle.New()
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
	appendVerifyRecord(t, b, "ai.human.approval", map[string]any{
		"approval_id":          "appr-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approve",
		"justification_digest": "just-digest",
		"action_id":            "act-1",
	}, "2026-03-27T12:03:00Z")
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
	appendVerifyRecord(
		t,
		b,
		bundle.AnchorEventType,
		`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:06:30Z"}`,
		"2026-03-27T12:06:30Z",
	)

	return b
}

func newSignedPrivilegedToolActionBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := bundle.New()
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

	policyFields := map[string]any{
		"policy_id":             "pol-1",
		"policy_version":        "2026-03",
		"decision":              "allow",
		"decision_reason_codes": []any{"ticket_present"},
		"subject_id_hash":       "subject-hash",
		"action_id":             "act-1",
	}
	signPolicyDecisionFields(t, policyFields)
	appendVerifyRecord(t, b, "ai.policy.decision", policyFields, "2026-03-27T12:02:00Z")

	appendVerifyRecord(t, b, "ai.human.approval", map[string]any{
		"approval_id":          "appr-1",
		"approver_id_hash":     "approver-hash",
		"approval_outcome":     "approve",
		"justification_digest": "just-digest",
		"action_id":            "act-1",
	}, "2026-03-27T12:03:00Z")
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
	appendVerifyRecord(
		t,
		b,
		bundle.AnchorEventType,
		`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:06:30Z"}`,
		"2026-03-27T12:06:30Z",
	)

	return b
}

func signPolicyDecisionFields(t testing.TB, fields map[string]any) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 keypair: %v", err)
	}

	signature, err := signpkg.SignPolicyDecision(fields, privateKey)
	if err != nil {
		t.Fatalf("sign policy decision: %v", err)
	}

	fields[event.FieldPolicySignature] = signature
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)
}

func hasInformationalNote(notes []string, want string) bool {
	for _, note := range notes {
		if note == want {
			return true
		}
	}
	return false
}

func hasRequiredWarning(warnings []string, want string) bool {
	for _, warning := range warnings {
		if warning == want {
			return true
		}
	}
	return false
}

func newVerifyBundleFromEvents(t testing.TB, events []event.Event) *bundle.Bundle {
	t.Helper()

	records := make([]bundle.Record, len(events))
	prevHash := hash.GenesisHash
	for i, evt := range events {
		evt.PrevHash = prevHash
		if evt.HashAlgo == "" {
			evt.HashAlgo = "sha256"
		}
		hashValue, err := hash.Compute(evt)
		if err != nil {
			t.Fatalf("compute hash for record %d: %v", i, err)
		}
		records[i] = bundle.Record{Event: evt, Hash: hashValue}
		prevHash = hashValue
	}

	return &bundle.Bundle{Records: records}
}
