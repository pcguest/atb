package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
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
					if data["public_key"] == "" {
						t.Fatalf("expected non-empty public_key field")
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

	b := bundle.New()
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
