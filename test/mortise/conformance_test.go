//go:build integration

package mortise_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
	"github.com/pcguest/atb/pkg/custody"
)

// TestMortiseIngestConformance asserts that passing profile fixtures produce
// stable verify.report.v1 output suitable for Mortise receipt storage.
func TestMortiseIngestConformance(t *testing.T) {
	schemaFields := verifyReportSchemaFields(t)
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-pass.atb") {
			continue
		}
		name := entry.Name()
		profileID := "atb.profile." + strings.TrimSuffix(name, "-pass.atb")

		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, name)
			b, err := bundle.Load(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if err := b.Verify(); err != nil {
				t.Fatalf("integrity: %v", err)
			}

			profile := verify.ProfileByID(profileID)
			if profile == nil {
				t.Fatalf("unknown profile %q", profileID)
			}

			report := verify.VerifyWithProfile(b, path, profile)
			vr := verify.ReportFromVerify(report)

			if vr.ReportVersion != verify.VerifyReportVersion {
				t.Fatalf("report_version = %q want %q", vr.ReportVersion, verify.VerifyReportVersion)
			}
			if vr.ProfileID != profileID {
				t.Fatalf("profile_id = %q want %q", vr.ProfileID, profileID)
			}
			if !vr.Pass {
				t.Fatalf("expected pass=true for -pass fixture, failures=%+v", vr.Failures)
			}
			if !vr.GateResult.ChainValid {
				t.Fatalf("expected chain_valid=true")
			}
			if vr.CASGrade == "" {
				t.Fatalf("expected cas_grade for CAS-supported profile")
			}
			if len(vr.Exclusions) == 0 {
				t.Fatalf("expected profile blind spots in exclusions")
			}
			if vr.ResidualRisk.Level == "" {
				t.Fatalf("expected residual_risk.level")
			}

			data, err := json.Marshal(vr)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var round map[string]json.RawMessage
			if err := json.Unmarshal(data, &round); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			for _, key := range []string{
				"report_version", "profile_id", "profile_version", "pass", "gate_result",
				"cas_score", "cas_grade", "sub_scores", "critical_failures", "residual_risk",
			} {
				if _, ok := round[key]; !ok {
					t.Errorf("custody contract missing field %q", key)
				}
			}
			for key := range round {
				if _, ok := schemaFields[key]; !ok {
					t.Errorf("custody contract emitted field %q outside %s", key, custody.VerifyReportSchemaVersion)
				}
			}
		})
	}
}

func verifyReportSchemaFields(t *testing.T) map[string]struct{} {
	t.Helper()
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(custody.VerifyReportSchemaJSON(), &schema); err != nil {
		t.Fatalf("unmarshal verify report schema: %v", err)
	}
	if len(schema.Properties) == 0 {
		t.Fatal("verify report schema has no properties")
	}
	fields := make(map[string]struct{}, len(schema.Properties))
	for key := range schema.Properties {
		fields[key] = struct{}{}
	}
	return fields
}
