//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
)

func TestProfileFixtures_PassAndFail(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	cases := []struct {
		file    string
		profile string
		pass    bool
	}{
		{"privileged_tool_action-pass.atb", profileIDPrivilegedToolAction, true},
		{"privileged_tool_action-fail.atb", profileIDPrivilegedToolAction, false},
		{"rag_answer-pass.atb", profileIDRAGAnswer, true},
		{"rag_answer-fail.atb", profileIDRAGAnswer, false},
		{"data_export-pass.atb", profileIDDataExport, true},
		{"data_export-fail.atb", profileIDDataExport, false},
		{"policy_decision-pass.atb", profileIDPolicyDecision, true},
		{"policy_decision-fail.atb", profileIDPolicyDecision, false},
		{"human_override-pass.atb", profileIDHumanOverride, true},
		{"human_override-fail.atb", profileIDHumanOverride, false},
		{"background_automation-pass.atb", profileIDBackgroundAutomation, true},
		{"background_automation-fail.atb", profileIDBackgroundAutomation, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			path := filepath.Join(root, tc.file)
			b, err := bundle.Load(path)
			if err != nil {
				t.Fatalf("load bundle: %v", err)
			}
			if err := b.Verify(); err != nil {
				t.Fatalf("integrity verify: %v", err)
			}
			profile := mustProfile(t, tc.profile)
			result := verify.VerifyWithProfile(b, path, profile)
			if len(result.Profiles) != 1 {
				t.Fatalf("expected 1 profile result, got %d", len(result.Profiles))
			}
			gotPass := result.Profiles[0].Pass
			if gotPass != tc.pass {
				t.Fatalf("expected pass=%v got pass=%v failures=%+v", tc.pass, gotPass, result.Profiles[0].CriticalFailures)
			}

			if tc.pass {
				vr := verify.ReportFromVerify(result)
				if vr.CASGrade == "" {
					t.Fatal("expected cas_grade for passing fixture")
				}
				if vr.CASScore <= 0 {
					t.Fatalf("expected positive cas_score, got %f", vr.CASScore)
				}
				if len(vr.SubScores) != 8 {
					t.Fatalf("expected 8 sub_scores keys, got %d", len(vr.SubScores))
				}
			} else {
				vr := verify.ReportFromVerify(result)
				if vr.Pass {
					t.Fatal("expected pass=false in stable report for -fail fixture")
				}
				if len(vr.Failures) == 0 {
					t.Fatal("expected critical_failures for -fail fixture")
				}
			}
		})
	}
}
