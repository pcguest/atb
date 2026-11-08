// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/v2/internal/bundle"
	"github.com/pcguest/atb/v2/internal/event"
	verifypkg "github.com/pcguest/atb/v2/internal/verify"
)

// These exercise the CLI handler with real on-disk bundles, rather than a
// constructed report. Process-level smoke checks are recorded separately.
func TestRunVerify_TruthfulEvidenceLanguage(t *testing.T) {
	cases := []struct {
		name          string
		signature     string
		anchor        bool
		tamper        bool
		profile       bool
		requireAnchor bool
		wantCode      int
		want          []string
		absent        []string
	}{
		{name: "unsigned without profile", wantCode: 0,
			want:   []string{"Integrity: PASS", "Profile: none matched", "Signature present: no", "Verification: NOT EVALUATED"},
			absent: []string{"✓ atb.bundle.signature", "Signature verified: yes"}},
		{name: "valid signature", signature: "valid", wantCode: 0,
			want: []string{"Integrity: PASS", "Signature present: yes", "Signature verified: yes"}},
		{name: "invalid signature", signature: "invalid", wantCode: 2,
			want:   []string{"Integrity: PASS", "Signature present: yes", "Signature verified: no", "signature mismatch", "Verification: FAIL"},
			absent: []string{"Signature verified: yes", "✓ atb.bundle.signature"}},
		{name: "tampered chain", tamper: true, wantCode: 2,
			want: []string{"Integrity: FAIL", "Verification: FAIL"}, absent: []string{"Integrity: PASS"}},
		{name: "absent optional anchor", wantCode: 0,
			want: []string{"anchor: not present"}, absent: []string{"anchor: failed"}},
		{name: "present failed anchor", anchor: true, wantCode: 0,
			want: []string{"anchor: failed", "Reason:"}, absent: []string{"anchor: not present"}},
		{name: "required signature event missing", profile: true, wantCode: 3,
			want:   []string{"Signature present: no", "signature event required", "[CRITICAL FAIL]", "Verification: FAIL"},
			absent: []string{"✓ atb.bundle.signature", "Signature verified: yes"}},
		{name: "required anchor event missing", profile: true, requireAnchor: true, wantCode: 3,
			want:   []string{"anchor: not present", "anchor event required", "[CRITICAL FAIL]", "Verification: FAIL"},
			absent: []string{"optional", "anchor: failed"}},
		{name: "source commitment without cryptographic signature", wantCode: 0,
			want:   []string{"SC (Source Commitment): 0.25", "- Source commitment evidence contributes to CAS", "No profile-specific obligations evaluated.", "Signature present: no"},
			absent: []string{"✓ atb.bundle.signature", "Signature verified: yes"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newTestBundle(t)
			stabiliseTestBundleManifest(t, b)
			if tc.tamper {
				appendTestBundleEvent(t, b, event.TypeDevSession, map[string]any{"value": "original"})
				b.Records[1].Event.Data = map[string]any{"value": "modified"}
			}
			if tc.anchor {
				appendTestBundleEvent(t, b, bundle.AnchorEventType, `{"bundle_hash":"invalid","tsr_der":"not-base64"}`)
			}
			path := writeVerifyTestBundle(t, b)
			if tc.signature != "" {
				keyPath := writeSignTestPrivateKey(t, t.TempDir())
				var out, errOut bytes.Buffer
				if code := runSign([]string{"--bundle", path, "--key", keyPath}, &out, &errOut); code != 0 {
					t.Fatalf("sign exit=%d: %s", code, errOut.String())
				}
				if tc.signature == "invalid" {
					signed, err := bundle.Load(path)
					if err != nil {
						t.Fatal(err)
					}
					for i := range signed.Records {
						if signed.Records[i].Event.Type == event.TypeBundleSignature {
							data := signed.Records[i].Event.Data.(map[string]any)
							data["signature"] = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x7f}, ed25519.SignatureSize))
						}
					}
					rehashTestBundle(t, signed)
					if err := signed.Save(path); err != nil {
						t.Fatal(err)
					}
				}
			}
			args := []string{"--bundle", path}
			if tc.profile {
				profilePath := filepath.Join(t.TempDir(), "signature.yaml")
				const profile = `id: org.example.signature_required
version: 1
workflow_class: signature_required
weights:
  EC: 0.20
  FC: 0.15
  RC: 0.20
  TC: 0.05
  SC: 0.10
  XC: 0.10
  AC: 0.10
  GC: 0.10
required:
  - type: atb.bundle.signature
    fields: []
    severity: critical
    message: signature event required
`
				profileText := profile
				if tc.requireAnchor {
					profileText = strings.ReplaceAll(profileText, "signature", "anchor")
				}
				if err := os.WriteFile(profilePath, []byte(profileText), 0600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--profile", profilePath)
			}
			var stdout, stderr bytes.Buffer
			if code := runVerify(args, &stdout, &stderr); code != tc.wantCode {
				t.Fatalf("exit=%d want=%d\n%s\n%s", code, tc.wantCode, stdout.String(), stderr.String())
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("missing %q:\n%s", want, stdout.String())
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(stdout.String(), absent) {
					t.Errorf("unexpected %q:\n%s", absent, stdout.String())
				}
			}
			// The same evidence must retain its structured-report verdict and exit.
			stdout.Reset()
			stderr.Reset()
			if code := runVerify(append(args, "--json"), &stdout, &stderr); code != tc.wantCode {
				t.Fatalf("JSON exit=%d want=%d: %s", code, tc.wantCode, stderr.String())
			}
			var report verifypkg.Report
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if tc.signature == "valid" && (report.BundleSignature == nil || !report.BundleSignature.Verified) {
				t.Fatal("valid signature not established by verifier")
			}
			if tc.signature == "" && report.BundleSignature != nil {
				t.Fatal("unsigned fixture has signature result")
			}
			if tc.name == "source commitment without cryptographic signature" && (report.CAS == nil || report.CAS.SubScores["SC"] != 0.25) {
				t.Fatal("fixture does not establish source commitment")
			}
		})
	}
}

func TestRenderVerifyTerminalReport_RequiredAnchorAbsent(t *testing.T) {
	var out bytes.Buffer
	renderVerifyTerminalReport(&out, verifypkg.Report{Anchoring: verifypkg.AnchoringResult{
		AnchorRequired: true, Summary: "anchor: failed (anchor record not present)",
	}})
	if !strings.Contains(out.String(), "anchor: not present (required)") || strings.Contains(out.String(), "anchor: failed") {
		t.Fatalf("inaccurate absent required anchor language: %s", out.String())
	}
}

func TestObligationPassText_SignatureEventIsNotCryptographicVerification(t *testing.T) {
	for _, warning := range []bool{false, true} {
		text := obligationPassText(verifyObligationSpec{eventType: event.TypeBundleSignature, warning: warning})
		if !strings.Contains(text, "event recorded; see Bundle Signature for cryptographic verification") {
			t.Fatalf("signature obligation overclaims: %s", text)
		}
	}
}

func TestRunVerify_FailedExplicitAnchorAttempt(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, bundle.AnchorEventType, `{"bundle_hash":"invalid","tsr_hash":"invalid"}`)
	path := writeVerifyTestBundle(t, b)
	if err := os.WriteFile(path+".tsr", []byte("invalid token"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := []string{"--bundle", path, "--with-anchor"}
			if format == "json" {
				args = append(args, "--json")
			}
			if code := runVerify(args, &stdout, &stderr); code != 2 {
				t.Fatalf("exit=%d want=2: %s", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), "anchor token hash mismatch") {
				t.Fatalf("missing failure reason: %s", stderr.String())
			}
		})
	}
}
