package main

import (
	"bytes"
	"encoding/json"
	"testing"

	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestVerifyJSONSnapshot(t *testing.T) {
	for _, tc := range verifySnapshotCases() {
		tc := tc
		name := tc.name
		if name == "" {
			name = tc.profileID
		}

		t.Run(name, func(t *testing.T) {
			bundlePath := buildSnapshotBundle(t, tc.events)
			if tc.afterBuild != nil {
				tc.afterBuild(t, bundlePath)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			args := append([]string{"--bundle", bundlePath, "--format", "json"}, tc.verifyArgs...)
			exitCode := runVerify(args, &stdout, &stderr)
			if exitCode != tc.wantVerifyExitCode {
				t.Errorf("runVerify() exit code = %d, want %d (stderr=%q)", exitCode, tc.wantVerifyExitCode, stderr.String())
			}

			var report verifypkg.VerifierReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Errorf("unmarshal verifier report: %v\noutput=%s", err, stdout.String())
				return
			}

			if tc.assertVerifyReport != nil {
				tc.assertVerifyReport(t, report)
				return
			}

			assertDefaultVerifyReportSnapshot(t, tc, report)
		})
	}
}

func assertDefaultVerifyReportSnapshot(t *testing.T, tc snapshotProfileCase, report verifypkg.VerifierReport) {
	t.Helper()

	if report.BundlePath == "" {
		t.Errorf("BundlePath is empty")
	}
	if report.ProfileID != tc.profileID {
		t.Errorf("ProfileID = %q, want %q", report.ProfileID, tc.profileID)
	}
	if !report.Pass {
		t.Errorf("Pass = false, want true")
	}
	if report.ResidualRisk == "" {
		t.Errorf("ResidualRisk is empty")
	}
	if len(report.Failures) != 0 {
		t.Errorf("Failures = %+v, want none", report.Failures)
	}
	if tc.expectCAS && report.CASGrade == "" {
		t.Errorf("CASGrade is empty")
	}
	if tc.expectCAS && report.CASScore <= 0 {
		t.Errorf("CASScore = %f, want > 0", report.CASScore)
	}
}
