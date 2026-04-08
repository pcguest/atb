package main

import (
	"bytes"
	"encoding/json"
	"testing"

	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestVerifyJSONSnapshot(t *testing.T) {
	for _, tc := range snapshotProfileCases() {
		tc := tc
		t.Run(tc.profileID, func(t *testing.T) {
			bundlePath := buildSnapshotBundle(t, tc.events)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runVerify([]string{"--bundle", bundlePath, "--format", "json"}, &stdout, &stderr)
			if exitCode != exitSuccess {
				t.Errorf("runVerify() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
			}

			var report verifypkg.VerifierReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Errorf("unmarshal verifier report: %v\noutput=%s", err, stdout.String())
				return
			}

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
		})
	}
}
