// SPDX-License-Identifier: MIT
package evidencepack

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderMarkdownPassAndErrorBundles(t *testing.T) {
	pack := Pack{
		Bundles: []BundleEvidenceSummary{
			{
				BundlePath:     "/tmp/pass.atb",
				ProfileID:      "atb.profile.rag_answer",
				ProfileVersion: 1,
				HeadHash:       "abc123",
				IntegrityPass:  true,
				ProfilePass:    true,
				CASScore:       0.87,
				CASGrade:       "High",
				ResidualRisk: &ResidualRiskSummary{
					Level:                   "Medium",
					Drivers:                 []string{"missing policy signature"},
					RecommendedNextEvidence: []string{"attach signed policy document"},
				},
				Exclusions: []string{"external retrieval cache"},
			},
			{
				BundlePath: "/tmp/missing.atb",
				Error:      "open /tmp/missing.atb: no such file or directory",
			},
		},
	}

	var buf bytes.Buffer
	fixedNow := time.Date(2026, 5, 25, 7, 0, 0, 0, time.UTC)
	if err := RenderMarkdown(pack, &buf, fixedNow, "1.12.0"); err != nil {
		t.Fatalf("RenderMarkdown() error = %v", err)
	}

	out := buf.String()
	wantSubstrings := []string{
		"# ATB evidence pack",
		"Generated at: 2026-05-25T07:00:00Z",
		"ATB version: v1.12.0",
		"## Bundle: /tmp/pass.atb",
		"- Profile: `atb.profile.rag_answer` (v1)",
		"- Head hash: `abc123`",
		"- Integrity: PASS",
		"- Profile obligations: PASS",
		"- CAS: 0.87 (High)",
		"### Residual risk",
		"- Level: Medium",
		"- Drivers:",
		"missing policy signature",
		"- Recommended next evidence:",
		"attach signed policy document",
		"### Exclusions",
		"external retrieval cache",
		"## Bundle: /tmp/missing.atb",
		"**Error:** open /tmp/missing.atb: no such file or directory",
		"## Notes for AI governance / AI Act Article 12",
		"Integrity PASS + Profile PASS + CAS High",
		"Profile FAIL or CAS Low",
		"Exclusions and residual risk drivers",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(out, want) {
			t.Fatalf("RenderMarkdown() missing %q in output:\n%s", want, out)
		}
	}
}
