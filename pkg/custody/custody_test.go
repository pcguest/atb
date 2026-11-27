// SPDX-License-Identifier: MIT
package custody_test

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pcguest/atb/v2/internal/bundle"
	"github.com/pcguest/atb/v2/internal/verify"
	"github.com/pcguest/atb/v2/pkg/custody"
)

// TestEvaluateMatchesFullVerifyPath asserts that custody.Evaluate produces the
// same verify.report.v1 document as the full evaluator path used by
// `atb verify`: verify.EvaluateBundle followed by
// verify.ReportFromVerifyWithBundle.
func TestEvaluateMatchesFullVerifyPath(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")

	evaluate := func(t *testing.T, name string) (diverged bool) {
		t.Helper()
		path := filepath.Join(root, name)
		profileName := strings.TrimSuffix(strings.TrimSuffix(name, ".atb"), "-pass")
		profileName = strings.TrimSuffix(profileName, "-fail")
		profileID := "atb.profile." + profileName

		profile := verify.ProfileByID(profileID)
		if profile == nil {
			t.Fatalf("unknown profile %q for fixture %s", profileID, name)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		b, err := bundle.Load(path)
		if err != nil {
			t.Fatalf("load bundle: %v", err)
		}

		got, err := custody.Evaluate(data, profileID, path)
		if err != nil {
			t.Fatalf("custody.Evaluate: %v", err)
		}

		report, err := verify.EvaluateBundle(verify.EvaluateConfig{
			BundlePath: path,
			Records:    b.Records,
			Profiles:   []verify.Profile{profile},
		})
		if err != nil {
			t.Fatalf("verify.EvaluateBundle: %v", err)
		}
		want := verify.ReportFromVerifyWithBundle(*report, b)

		// The evaluator accumulates the CAS weighted total in Go map iteration
		// order, so two evaluations of identical records can differ in the
		// final float bit. Compare everything exactly after normalising those
		// summation-order artefacts.
		if !reflect.DeepEqual(normaliseCASNumbers(got.Report), normaliseCASNumbers(want)) {
			t.Fatalf("custody.Evaluate report differs from full evaluator\ngot:  %s\nwant: %s",
				marshalReport(t, got.Report), marshalReport(t, want))
		}
		return len(got.Report.ProvabilityGaps) > 0 || len(got.Report.ResidualRisk.Drivers) > 0
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}
	var passFixtures, failFixtures []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".atb") {
			continue
		}
		switch {
		case strings.HasSuffix(entry.Name(), "-pass.atb"):
			passFixtures = append(passFixtures, entry.Name())
		case strings.HasSuffix(entry.Name(), "-fail.atb"):
			failFixtures = append(failFixtures, entry.Name())
		}
	}
	if len(passFixtures) == 0 {
		t.Fatal("no -pass.atb profile fixtures found")
	}

	divergenceProven := false
	for _, name := range passFixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			if evaluate(t, name) {
				divergenceProven = true
			}
		})
	}
	if divergenceProven {
		return
	}

	if len(failFixtures) == 0 {
		t.Fatal("no fixture exercised a non-trivial divergence: pass fixtures reported no provability_gaps or residual_risk.drivers, and no -fail.atb fixture exists")
	}
	t.Run("divergence:"+failFixtures[0], func(t *testing.T) {
		if !evaluate(t, failFixtures[0]) {
			t.Fatalf("fixture %s exercised no non-trivial divergence: provability_gaps and residual_risk.drivers are both empty", failFixtures[0])
		}
	})
}

func marshalReport(t *testing.T, report custody.VerifierReport) string {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return string(data)
}

// normaliseCASNumbers returns a deep copy of report with the CAS totals and
// sub-scores rounded to nine decimal places. The evaluator sums weighted
// dimensions in Go map iteration order, so independent evaluations of identical
// records can differ in the final bits of cas_score. Every other field is left
// untouched and compared exactly.
func normaliseCASNumbers(report custody.VerifierReport) custody.VerifierReport {
	out := report
	out.CASScore = roundCASValue(report.CASScore)
	out.EffectiveScore = roundCASValue(report.EffectiveScore)
	out.CorroborationBonus = roundCASValue(report.CorroborationBonus)
	if report.SubScores != nil {
		out.SubScores = make(map[string]float64, len(report.SubScores))
		for key, value := range report.SubScores {
			out.SubScores[key] = roundCASValue(value)
		}
	}

	out.Failures = append(report.Failures[:0:0], report.Failures...)
	out.Obligations = append(report.Obligations[:0:0], report.Obligations...)
	out.Warnings = append(report.Warnings[:0:0], report.Warnings...)
	out.Notes = append(report.Notes[:0:0], report.Notes...)
	out.Exclusions = append(report.Exclusions[:0:0], report.Exclusions...)
	out.Signatures = append(report.Signatures[:0:0], report.Signatures...)
	out.ProvabilityGaps = append(report.ProvabilityGaps[:0:0], report.ProvabilityGaps...)
	out.ReviewerIdentities = append(report.ReviewerIdentities[:0:0], report.ReviewerIdentities...)

	out.ResidualRisk = report.ResidualRisk
	out.ResidualRisk.Drivers = append(report.ResidualRisk.Drivers[:0:0], report.ResidualRisk.Drivers...)
	out.ResidualRisk.RecommendedNextEvidence = append(report.ResidualRisk.RecommendedNextEvidence[:0:0], report.ResidualRisk.RecommendedNextEvidence...)

	if report.GateResult.AnchorPass != nil {
		anchorPass := *report.GateResult.AnchorPass
		out.GateResult.AnchorPass = &anchorPass
	}

	if report.DimensionAssessments != nil {
		out.DimensionAssessments = make(map[string]verify.DimensionAssessment, len(report.DimensionAssessments))
		for key, assessment := range report.DimensionAssessments {
			if assessment.Score != nil {
				score := *assessment.Score
				assessment.Score = &score
			}
			out.DimensionAssessments[key] = assessment
		}
	}

	return out
}

func roundCASValue(value float64) float64 {
	const precision = 1e9
	return math.Round(value*precision) / precision
}

// TestAnchorRequiredEvaluatorDivergence demonstrates the evaluator-level
// difference between verify.Verify and verify.EvaluateBundle: only the latter
// accepts EvaluateConfig.AnchorRequired. With no verified timestamp anchor, the
// full evaluator applies applyRequiredAnchorRisk and demotes assurance and
// residual risk; verify.Verify cannot express that policy. custody.Evaluate
// exposes no anchor option, so this divergence is not reachable through current
// custody inputs.
func TestAnchorRequiredEvaluatorDivergence(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	path := filepath.Join(root, "rag_answer-pass.atb")
	b, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	profile := verify.ProfileByID("atb.profile.rag_answer")
	if profile == nil {
		t.Fatal("rag_answer profile not registered")
	}

	legacy := verify.Verify(b, path, profile.ID())
	if legacy.Anchoring.TSAVerified {
		t.Fatal("fixture unexpectedly carries a verified timestamp anchor")
	}
	if legacy.Anchoring.AnchorRequired {
		t.Fatal("verify.Verify unexpectedly set anchor_required")
	}
	if legacy.CAS == nil {
		t.Fatal("verify.Verify produced no CAS result")
	}
	if !legacy.CAS.AssuranceValid {
		t.Fatal("verify.Verify unexpectedly demoted assurance_valid")
	}
	if legacy.ResidualRisk.Level == "High" || legacy.ResidualRisk.Level == "Critical" {
		t.Fatalf("fixture baseline residual_risk.level = %q, already demoted", legacy.ResidualRisk.Level)
	}
	if slices.Contains(legacy.ResidualRisk.Drivers, "required_anchor_unverified") {
		t.Fatal("verify.Verify unexpectedly applied required-anchor risk")
	}

	full, err := verify.EvaluateBundle(verify.EvaluateConfig{
		BundlePath:     path,
		Records:        b.Records,
		Profiles:       []verify.Profile{profile},
		AnchorRequired: true,
	})
	if err != nil {
		t.Fatalf("verify.EvaluateBundle: %v", err)
	}
	if !full.Anchoring.AnchorRequired {
		t.Fatal("EvaluateBundle did not apply anchor_required")
	}
	if full.CAS == nil {
		t.Fatal("EvaluateBundle produced no CAS result")
	}
	if full.CAS.AssuranceValid {
		t.Fatal("EvaluateBundle with AnchorRequired did not demote assurance_valid")
	}
	if full.ResidualRisk.Level != "High" {
		t.Fatalf("residual_risk.level = %q, want High", full.ResidualRisk.Level)
	}
	if !slices.Contains(full.ResidualRisk.Drivers, "required_anchor_unverified") {
		t.Fatalf("residual_risk.drivers = %v, want required_anchor_unverified", full.ResidualRisk.Drivers)
	}
	if reflect.DeepEqual(legacy.ResidualRisk, full.ResidualRisk) {
		t.Fatal("expected EvaluateBundle residual risk to differ from verify.Verify")
	}
}

func TestEvaluatePassFixture(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	path := filepath.Join(root, "rag_answer-pass.atb")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := custody.Evaluate(data, "atb.profile.rag_answer", path)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.HeadHash == "" {
		t.Fatal("expected non-empty head hash")
	}
	if result.Report.ReportVersion != custody.VerifyReportVersion {
		t.Fatalf("report_version = %q want %q", result.Report.ReportVersion, custody.VerifyReportVersion)
	}
	if !result.Report.Pass {
		t.Fatalf("expected pass=true, failures=%+v", result.Report.Failures)
	}
	if result.Report.CASGrade == "" {
		t.Fatal("expected cas_grade")
	}

	raw, err := json.Marshal(result.Report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]json.RawMessage
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{
		"report_version", "profile_id", "pass", "gate_result",
		"cas_score", "cas_grade", "sub_scores", "critical_failures", "residual_risk",
	} {
		if _, ok := round[key]; !ok {
			t.Errorf("custody contract missing field %q", key)
		}
	}
}

func TestEvaluateFailFixture(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	path := filepath.Join(root, "rag_answer-fail.atb")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := custody.Evaluate(data, "atb.profile.rag_answer", path)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.Report.Pass {
		t.Fatal("expected pass=false for -fail fixture")
	}
	if len(result.Report.Failures) == 0 {
		t.Fatal("expected critical failures")
	}
}

func TestEvaluateNormalisesProfileID(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	path := filepath.Join(root, "policy_decision-pass.atb")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := custody.Evaluate(data, "policy_decision", path)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !strings.HasPrefix(result.Report.ProfileID, "atb.profile.") {
		t.Fatalf("profile_id = %q", result.Report.ProfileID)
	}
}

func TestLoadBundleRejectsEmpty(t *testing.T) {
	_, err := custody.LoadBundle(nil)
	if err == nil {
		t.Fatal("expected error for empty bundle")
	}
}

func TestEvaluateRejectsUnknownProfile(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	data, err := os.ReadFile(filepath.Join(root, "rag_answer-pass.atb"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	for _, profileID := range []string{"atb.profile.not_registered", "not_registered"} {
		profileID := profileID
		t.Run(profileID, func(t *testing.T) {
			_, err := custody.Evaluate(data, profileID, "")
			if err == nil {
				t.Fatalf("expected error for unknown profile %q", profileID)
			}
			if !errors.Is(err, verify.ErrProfileUnknown) {
				t.Fatalf("expected error wrapping verify.ErrProfileUnknown, got: %v", err)
			}
		})
	}
}
