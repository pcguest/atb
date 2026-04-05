package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestRunSign(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "sign_creates_signature_record",
			run: func(t *testing.T) {
				b := buildSignSCBundle(t)
				bundlePath := writeVerifyTestBundle(t, b)
				keyPath := writeSignTestPrivateKey(t, t.TempDir())

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				exitCode := runSign([]string{"--bundle", bundlePath, "--key", keyPath}, &stdout, &stderr)
				if exitCode != exitSuccess {
					t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
				}

				signedBundle, err := bundle.Load(bundlePath)
				if err != nil {
					t.Fatalf("load signed bundle: %v", err)
				}

				signatureCount := 0
				for _, record := range signedBundle.Records {
					if record.Event.Type != event.TypeBundleSignature {
						continue
					}
					signatureCount++

					data, ok := record.Event.Data.(map[string]any)
					if !ok {
						t.Fatalf("signature record data type = %T, want map[string]any", record.Event.Data)
					}
					if data["signature"] == "" {
						t.Fatalf("expected non-empty signature field")
					}
					if data["pubkey"] == "" {
						t.Fatalf("expected non-empty pubkey field")
					}
					if data["public_key"] != nil {
						t.Fatalf("expected signature record to use pubkey, got legacy public_key field")
					}
				}
				if signatureCount != 1 {
					t.Fatalf("expected exactly one signature record, got %d", signatureCount)
				}
			},
		},
		{
			name: "sign_missing_bundle",
			run: func(t *testing.T) {
				keyPath := writeSignTestPrivateKey(t, t.TempDir())

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				exitCode := runSign([]string{"--bundle", filepath.Join(t.TempDir(), "missing.atb"), "--key", keyPath}, &stdout, &stderr)
				if exitCode == exitSuccess {
					t.Fatalf("expected sign to fail for missing bundle")
				}
			},
		},
		{
			name: "sign_missing_key",
			run: func(t *testing.T) {
				bundlePath := writeVerifyTestBundle(t, buildSignSCBundle(t))

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				exitCode := runSign([]string{"--bundle", bundlePath, "--key", filepath.Join(t.TempDir(), "missing-key.pem")}, &stdout, &stderr)
				if exitCode == exitSuccess {
					t.Fatalf("expected sign to fail for missing key")
				}
			},
		},
		{
			name: "sign_then_verify_signature_round_trip",
			run: func(t *testing.T) {
				bundlePath := writeVerifyTestBundle(t, buildSignSCBundle(t))
				keyDir := t.TempDir()

				var keygenStdout bytes.Buffer
				var keygenStderr bytes.Buffer
				exitCode := runKeygen([]string{"--out-dir", keyDir}, &keygenStdout, &keygenStderr)
				if exitCode != exitSuccess {
					t.Fatalf("unexpected keygen exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, keygenStderr.String())
				}

				keyPath := filepath.Join(keyDir, "atb-key.pem")
				var signStdout bytes.Buffer
				var signStderr bytes.Buffer
				exitCode = runSign([]string{"--bundle", bundlePath, "--key", keyPath}, &signStdout, &signStderr)
				if exitCode != exitSuccess {
					t.Fatalf("unexpected sign exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, signStderr.String())
				}

				var verifyStdout bytes.Buffer
				var verifyStderr bytes.Buffer
				exitCode = runVerify([]string{"--bundle", bundlePath, "--json"}, &verifyStdout, &verifyStderr)
				if exitCode != exitSuccess {
					t.Fatalf("unexpected verify exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, verifyStderr.String())
				}

				var report verifypkg.Report
				if err := json.Unmarshal(verifyStdout.Bytes(), &report); err != nil {
					t.Fatalf("unmarshal verify report: %v", err)
				}
				if report.BundleSignature == nil {
					t.Fatalf("expected bundle signature result")
				}
				if !report.BundleSignature.Present || !report.BundleSignature.Verified {
					t.Fatalf("expected verified bundle signature, got %+v", *report.BundleSignature)
				}
			},
		},
		{
			name: "sign_then_verify_sc_uplift",
			run: func(t *testing.T) {
				b := buildSignSCBundle(t)
				bundlePath := writeVerifyTestBundle(t, b)
				keyPath := writeSignTestPrivateKey(t, t.TempDir())

				unsignedBundle, err := bundle.Load(bundlePath)
				if err != nil {
					t.Fatalf("load unsigned bundle: %v", err)
				}
				unsignedReport := verifypkg.Verify(unsignedBundle, bundlePath, "atb.profile.privileged_tool_action")
				if unsignedReport.CAS == nil {
					t.Fatalf("expected CAS for unsigned bundle")
				}

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				exitCode := runSign([]string{"--bundle", bundlePath, "--key", keyPath}, &stdout, &stderr)
				if exitCode != exitSuccess {
					t.Fatalf("unexpected exit code: got %d want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
				}

				signedBundle, err := bundle.Load(bundlePath)
				if err != nil {
					t.Fatalf("load signed bundle: %v", err)
				}
				signedReport := verifypkg.Verify(signedBundle, bundlePath, "atb.profile.privileged_tool_action")
				if signedReport.CAS == nil {
					t.Fatalf("expected CAS for signed bundle")
				}

				diff := signedReport.CAS.SubScores["SC"] - unsignedReport.CAS.SubScores["SC"]
				if diff < 0.099 || diff > 0.101 {
					t.Fatalf("unexpected SC uplift: got %.3f want about 0.100", diff)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func buildSignSCBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b := newTestBundle(t)
	appendTestBundleEventWithOptions(t, b, event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "actor-hash",
		"purpose_tag":   "approve-change",
	}, &bundle.AppendOptions{Timestamp: "2026-03-27T12:00:00Z"})
	appendTestBundleEventWithOptions(
		t,
		b,
		event.TypeBundleAnchor,
		`{"bundle_hash":"bundle-hash","tsr_hash":"tsr-hash","certified_time":"2026-03-27T12:01:30Z"}`,
		&bundle.AppendOptions{Timestamp: "2026-03-27T12:01:30Z"},
	)
	return b
}

func writeSignTestPrivateKey(t testing.TB, dir string) string {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 keypair: %v", err)
	}

	privatePEM, err := marshalEd25519PrivateKeyPEM(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	path := filepath.Join(dir, "atb-key.pem")
	if err := os.WriteFile(path, privatePEM, 0600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	return path
}
