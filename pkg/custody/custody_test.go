// SPDX-License-Identifier: MIT
package custody_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/pkg/custody"
)

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
