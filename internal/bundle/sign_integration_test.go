package bundle_test

import (
	"bytes"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/verify"
)

func TestBundleSignAndVerify(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := cryptorand.Read(seed); err != nil {
		t.Fatalf("generate Ed25519 seed: %v", err)
	}

	publicKey, privateKey, err := ed25519.GenerateKey(bytes.NewReader(seed))
	if err != nil {
		t.Fatalf("generate Ed25519 keypair: %v", err)
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	if err := b.AppendWithOptions(event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-123",
		"actor_id_hash": "actor-456",
		"purpose_tag":   "rag_answer",
	}, &bundle.AppendOptions{Timestamp: "2026-04-09T00:00:00Z"}); err != nil {
		t.Fatalf("append request event: %v", err)
	}

	if err := b.AppendWithOptions(event.TypeAIModelInvoked, map[string]any{
		"model_provider":           "openai",
		"model_id":                 "gpt-5.4",
		"model_parameters_digest":  "params-sha256",
		"prompt_digest":            "prompt-sha256",
	}, &bundle.AppendOptions{Timestamp: "2026-04-09T00:00:01Z"}); err != nil {
		t.Fatalf("append model event: %v", err)
	}

	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	if err := bundle.Sign(path, privateKey); err != nil {
		t.Fatalf("sign bundle: %v", err)
	}

	signedBundle, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load signed bundle: %v", err)
	}

	signatureRecord, signatureValue := findBundleSignatureRecord(t, signedBundle)
	signatureData, ok := signatureRecord.Event.Data.(map[string]any)
	if !ok {
		t.Fatalf("signature record data type = %T, want map[string]any", signatureRecord.Event.Data)
	}

	pubkeyValue, ok := signatureData["pubkey"].(string)
	if !ok || pubkeyValue == "" {
		t.Fatalf("signature record pubkey missing")
	}
	if got, want := pubkeyValue, base64.StdEncoding.EncodeToString(publicKey); got != want {
		t.Fatalf("signature record pubkey = %q, want %q", got, want)
	}

	report := verify.Verify(signedBundle, path, "")
	if !report.Integrity.ChainValid {
		t.Fatalf("expected chain_valid=true, got report=%+v", report.Integrity)
	}
	if report.Integrity.Error != "" {
		t.Fatalf("expected empty integrity error, got %q", report.Integrity.Error)
	}
	if report.BundleSignature == nil {
		t.Fatalf("expected bundle signature result")
	}
	if !report.BundleSignature.Present {
		t.Fatalf("expected bundle signature to be present")
	}
	if !report.BundleSignature.Verified {
		t.Fatalf("expected bundle signature to verify, got %+v", *report.BundleSignature)
	}
	if report.BundleSignature.Error != "" {
		t.Fatalf("expected empty bundle signature error, got %q", report.BundleSignature.Error)
	}

	corruptBundleSignature(t, path, signatureValue)

	corruptedBundle, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load corrupted bundle: %v", err)
	}

	corruptedReport := verify.Verify(corruptedBundle, path, "")
	if corruptedReport.Integrity.ChainValid && corruptedReport.BundleSignature != nil && corruptedReport.BundleSignature.Verified {
		t.Fatalf("expected corrupted bundle verification to fail, got report=%+v", corruptedReport)
	}
	if corruptedReport.Integrity.ChainValid {
		if corruptedReport.BundleSignature == nil || corruptedReport.BundleSignature.Error == "" || corruptedReport.BundleSignature.Verified {
			t.Fatalf("expected signature verification failure, got report=%+v", corruptedReport.BundleSignature)
		}
	} else if corruptedReport.Integrity.Error == "" {
		t.Fatalf("expected integrity error for corrupted bundle")
	}
	if corruptedReport.ResidualRisk.Level != "Critical" && corruptedReport.ResidualRisk.Level != "High" {
		t.Fatalf("unexpected residual risk level: got %q want Critical or High", corruptedReport.ResidualRisk.Level)
	}
}

func findBundleSignatureRecord(t testing.TB, b *bundle.Bundle) (bundle.Record, string) {
	t.Helper()

	for _, record := range b.Records {
		if record.Event.Type != event.TypeBundleSignature {
			continue
		}

		data, ok := record.Event.Data.(map[string]any)
		if !ok {
			t.Fatalf("signature record data type = %T, want map[string]any", record.Event.Data)
		}

		signatureValue, ok := data["signature"].(string)
		if !ok || signatureValue == "" {
			t.Fatalf("signature record signature missing")
		}

		return record, signatureValue
	}

	t.Fatal("expected atb.bundle.signature record")
	return bundle.Record{}, ""
}

func corruptBundleSignature(t testing.TB, path string, signatureValue string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat bundle: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}

	corruptedSignature := signatureValue[:len(signatureValue)-1] + "X"
	if corruptedSignature == signatureValue {
		corruptedSignature = signatureValue[:len(signatureValue)-1] + "Y"
	}

	corrupted := strings.Replace(string(raw), "\"signature\":\""+signatureValue+"\"", "\"signature\":\""+corruptedSignature+"\"", 1)
	if corrupted == string(raw) {
		t.Fatal("failed to corrupt signature value")
	}

	if err := os.WriteFile(path, []byte(corrupted), info.Mode().Perm()); err != nil {
		t.Fatalf("write corrupted bundle: %v", err)
	}
}
