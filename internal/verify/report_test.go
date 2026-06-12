// SPDX-License-Identifier: MIT
package verify

import (
	"encoding/json"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestVerifierReport_JSONInterfaceStability(t *testing.T) {
	report := VerifierReport{
		ReportVersion: VerifyReportVersion,
		BundlePath:    "bundle.atb",
		ProfileID:     profileIDPrivilegedToolAction,
		Pass:          true,
		GateResult: GateResult{
			Pass:           true,
			ChainValid:     true,
			ProfilePass:    true,
			AnchorRequired: false,
		},
		CASScore:  0.87,
		CASGrade:  "High",
		SubScores: map[string]float64{"EC": 1, "FC": 1, "RC": 1, "TC": 1, "SC": 0.8, "XC": 0, "AC": 0, "GC": 1},
		Failures:  []ReportFailure{},
		Warnings:  []string{},
		Notes:     []string{},
		ResidualRisk: ResidualRiskReport{
			Level: "Low",
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal VerifierReport: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	required := []string{
		"report_version",
		"bundle_path",
		"profile_id",
		"pass",
		"gate_result",
		"cas_score",
		"cas_grade",
		"sub_scores",
		"critical_failures",
		"required_warnings",
		"informational_notes",
		"residual_risk",
	}
	for _, key := range required {
		if _, ok := decoded[key]; !ok {
			t.Errorf("VerifierReport JSON missing required field %q", key)
		}
	}

	if got := string(decoded["report_version"]); got != `"verify.report.v1"` {
		t.Errorf("report_version = %s, want %q", got, VerifyReportVersion)
	}

	var subScores map[string]float64
	if err := json.Unmarshal(decoded["sub_scores"], &subScores); err != nil {
		t.Fatalf("unmarshal sub_scores: %v", err)
	}
	for _, code := range []string{"EC", "FC", "RC", "TC", "SC", "XC", "AC", "GC"} {
		if _, ok := subScores[code]; !ok {
			t.Errorf("sub_scores missing key %q", code)
		}
	}

	var risk ResidualRiskReport
	if err := json.Unmarshal(decoded["residual_risk"], &risk); err != nil {
		t.Fatalf("residual_risk must be object, got error: %v", err)
	}
	if risk.Level == "" {
		t.Error("residual_risk.level must be non-empty")
	}
}

func TestReportFromVerify_IncludesExclusions(t *testing.T) {
	b := newPrivilegedToolActionBundle(t)
	report := Verify(b, "bundle.atb", profileIDPrivilegedToolAction)
	vr := ReportFromVerify(report)

	if vr.ReportVersion != VerifyReportVersion {
		t.Errorf("ReportVersion = %q, want %q", vr.ReportVersion, VerifyReportVersion)
	}
	if vr.ProfileID != profileIDPrivilegedToolAction {
		t.Errorf("ProfileID = %q", vr.ProfileID)
	}
	if len(vr.Exclusions) == 0 {
		t.Error("expected profile blind spots in exclusions")
	}
	if vr.GateResult.ChainValid != true {
		t.Errorf("GateResult.ChainValid = %v", vr.GateResult.ChainValid)
	}
	if vr.CASGrade == "" {
		t.Error("expected CAS grade for CAS-supported profile")
	}
	if vr.ProfileVersion == 0 {
		t.Error("expected profile_version for built-in profile")
	}
	if len(vr.ResidualRisk.Drivers) == 0 && vr.CASScore < 0.85 {
		t.Log("note: no residual risk drivers despite CAS < 0.85 (all sub-scores may be >= 0.70)")
	}
}

func TestReportFromVerify_IncludesIntegrityFailure(t *testing.T) {
	report := Report{
		BundlePath: "tampered.atb",
		Integrity: IntegrityResult{
			ChainValid: false,
			Error:      "record hash mismatch at seq 2",
		},
		Profiles: []ProfileResult{{
			ProfileID: profileIDPrivilegedToolAction,
			Version:   1,
			CriticalFailures: []CriticalFailure{{
				ID:     "required:ai.action.precommit",
				Kind:   "missing_event",
				Detail: "ai.action.precommit missing required fields",
			}},
		}},
		ResidualRisk: ResidualRisk{
			Level: "Critical",
		},
	}

	vr := ReportFromVerify(report)
	if len(vr.Failures) != 2 {
		t.Fatalf("critical_failures = %+v, want integrity and profile failures", vr.Failures)
	}
	if got := vr.Failures[0]; got.ID != "integrity:chain" ||
		got.Kind != "integrity_failure" ||
		got.Detail != "record hash mismatch at seq 2" {
		t.Fatalf("critical_failures[0] = %+v", got)
	}
}

func TestReportFromVerify_PropagatesResidualRiskDrivers(t *testing.T) {
	b := newRAGAnswerBundle(t)
	filtered := make([]bundle.Record, 0, len(b.Records))
	for _, rec := range b.Records {
		if rec.Event.Type == event.TypeAIModelInvoked {
			continue
		}
		filtered = append(filtered, rec)
	}
	b.Records = filtered

	report := Verify(b, "bundle.atb", profileIDRAGAnswer)
	vr := ReportFromVerify(report)

	if vr.ResidualRisk.Level == "" {
		t.Fatal("expected residual_risk.level")
	}
	if len(vr.ResidualRisk.Drivers) == 0 {
		t.Fatalf("expected residual risk drivers, got %+v", vr.ResidualRisk)
	}
	if vr.ProfileVersion <= 0 {
		t.Errorf("profile_version = %d, want > 0", vr.ProfileVersion)
	}
}
