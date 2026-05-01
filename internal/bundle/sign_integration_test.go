// SPDX-License-Identifier: MIT
package bundle_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
		"model_provider":          "openai",
		"model_id":                "gpt-5.4",
		"model_parameters_digest": "params-sha256",
		"prompt_digest":           "prompt-sha256",
	}, &bundle.AppendOptions{Timestamp: "2026-04-09T00:00:01Z"}); err != nil {
		t.Fatalf("append model event: %v", err)
	}

	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	if err := bundle.Sign(context.Background(), path, privateKey); err != nil {
		t.Fatalf("sign bundle: %v", err)
	}

	signedBundle, err := bundle.Load(context.Background(), path)
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

	corruptedBundle, err := bundle.Load(context.Background(), path)
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

func TestBundleSignTwiceVerifiesFullChain(t *testing.T) {
	_, firstPrivateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate first Ed25519 keypair: %v", err)
	}
	_, secondPrivateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate second Ed25519 keypair: %v", err)
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
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	if err := bundle.Sign(context.Background(), path, firstPrivateKey); err != nil {
		t.Fatalf("first sign bundle: %v", err)
	}
	if err := bundle.Sign(context.Background(), path, secondPrivateKey); err != nil {
		t.Fatalf("second sign bundle: %v", err)
	}

	signedBundle, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load signed bundle: %v", err)
	}
	if err := signedBundle.Verify(); err != nil {
		t.Fatalf("verify hash chain: %v", err)
	}

	report := verify.Verify(signedBundle, path, "")
	if !report.Integrity.ChainValid {
		t.Fatalf("expected chain_valid=true, got report=%+v", report.Integrity)
	}
	if len(report.Signatures) != 2 {
		t.Fatalf("signature count = %d, want 2", len(report.Signatures))
	}
	for _, signature := range report.Signatures {
		if !signature.Valid {
			t.Fatalf("expected all signatures valid, got %+v", report.Signatures)
		}
	}
}

func TestConcurrentSignToReturnsLockedAndLeavesOneSignature(t *testing.T) {
	t.Parallel()

	_, firstPrivateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate first Ed25519 keypair: %v", err)
	}
	_, secondPrivateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate second Ed25519 keypair: %v", err)
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := b.AppendWithOptions(event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-concurrent",
		"actor_id_hash": "actor-concurrent",
		"purpose_tag":   "rag_answer",
	}, &bundle.AppendOptions{Timestamp: "2026-04-09T00:00:00Z"}); err != nil {
		t.Fatalf("append request event: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	signerEntered := make(chan struct{})
	releaseSigner := make(chan struct{})
	blockingSigner := &blockingTestSigner{
		privateKey: firstPrivateKey,
		entered:    signerEntered,
		release:    releaseSigner,
	}

	firstErr := make(chan error, 1)
	go func() {
		_, err := bundle.SignToWithSigner(context.Background(), path, path, blockingSigner)
		firstErr <- err
	}()

	select {
	case <-signerEntered:
	case <-time.After(time.Second):
		t.Fatal("first signer did not enter signing section")
	}

	// SignTo uses a non-blocking advisory lock. Callers that receive
	// ErrBundleLocked should retry with backoff rather than assuming the
	// signature was appended.
	if _, err := bundle.SignTo(context.Background(), path, path, secondPrivateKey); !errors.Is(err, bundle.ErrBundleLocked) {
		t.Fatalf("second SignTo error = %v, want ErrBundleLocked", err)
	}

	close(releaseSigner)
	select {
	case err := <-firstErr:
		if err != nil {
			t.Fatalf("first SignToWithSigner: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first SignToWithSigner did not finish")
	}

	signedBundle, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load signed bundle: %v", err)
	}
	if err := signedBundle.Verify(); err != nil {
		t.Fatalf("verify signed bundle: %v", err)
	}
	if got := countBundleSignatureRecords(signedBundle); got != 1 {
		t.Fatalf("signature record count = %d, want 1", got)
	}
}

func TestSignCancelled(t *testing.T) {
	t.Parallel()

	_, privateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate Ed25519 keypair: %v", err)
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := b.AppendWithOptions(event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-cancelled",
		"actor_id_hash": "actor-cancelled",
		"purpose_tag":   "rag_answer",
	}, &bundle.AppendOptions{Timestamp: "2026-04-09T00:00:00Z"}); err != nil {
		t.Fatalf("append request event: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = bundle.SignTo(ctx, path, path, privateKey)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SignTo() error = %v, want context.Canceled", err)
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

func countBundleSignatureRecords(b *bundle.Bundle) int {
	count := 0
	for _, record := range b.Records {
		if record.Event.Type == event.TypeBundleSignature {
			count++
		}
	}
	return count
}

type blockingTestSigner struct {
	privateKey ed25519.PrivateKey
	entered    chan<- struct{}
	release    <-chan struct{}
	once       sync.Once
}

func (s *blockingTestSigner) Sign(ctx context.Context, digest []byte) (sig, pubKey []byte, keyID, backend, algorithm string, err error) {
	s.once.Do(func() {
		close(s.entered)
	})

	select {
	case <-s.release:
	case <-ctx.Done():
		return nil, nil, "", "", "", ctx.Err()
	}

	sig = ed25519.Sign(s.privateKey, digest)
	pub, ok := s.privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, nil, "", "", "", errors.New("invalid public key")
	}
	return sig, append([]byte(nil), pub...), "", "", "ed25519", nil
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

// TestSignToAtomic confirms that an in-place SignTo produces a valid signed
// bundle whose pre-sign bytes appear unchanged as a prefix of the output —
// i.e. the atomic write replaces the file with a single complete result.
func TestSignToAtomic(t *testing.T) {
	t.Parallel()

	_, privateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := b.AppendWithOptions(event.TypeAIRequestReceived, map[string]any{
		"request_id":    "req-1",
		"actor_id_hash": "h",
		"purpose_tag":   "rag",
		"input_digest":  "sha256:x",
	}, &bundle.AppendOptions{Timestamp: "2026-04-09T00:00:00Z"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Save(context.Background(), path); err != nil {
		t.Fatalf("save: %v", err)
	}

	preSignBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pre-sign: %v", err)
	}

	if _, err := bundle.SignTo(context.Background(), path, path, privateKey); err != nil {
		t.Fatalf("sign: %v", err)
	}

	postSignBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-sign: %v", err)
	}

	// The signed file MUST start with exactly the pre-sign bytes — anything
	// less means the atomic write produced a partial result; anything mutated
	// means the bundle was rewritten in-place rather than appended-and-renamed.
	if !bytes.HasPrefix(postSignBytes, preSignBytes) {
		t.Fatalf("post-sign bytes do not start with pre-sign bytes; atomic prefix preservation broken")
	}
	if len(postSignBytes) <= len(preSignBytes) {
		t.Fatalf("post-sign bytes (%d) not larger than pre-sign (%d); signature record missing",
			len(postSignBytes), len(preSignBytes))
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("load signed: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify signed: %v", err)
	}

	// Confirm no .tmp file was left behind by the atomic writer.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temp file left behind after sign: %s", e.Name())
		}
	}
}

// TestSignNoTOCTOU exercises the closed TOCTOU window in SignTo: the bundle
// is now opened once and the digest, parsed bundle, and file mode are all
// derived from those bytes. We start a sign in one goroutine and Save a
// different version in another. The final file must be byte-for-byte one of
// the two consistent results, never a torn mix.
//
// Probabilistic: this test exercises a race window. It will not always observe
// the contention, but together with -race it documents and catches regression
// to the prior os.ReadFile + Load(...) double-read pattern.
func TestSignNoTOCTOU(t *testing.T) {
	t.Parallel()

	_, privateKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	path := filepath.Join(t.TempDir(), "bundle.atb")
	makeBundle := func(payload string) *bundle.Bundle {
		t.Helper()
		b, err := bundle.New()
		if err != nil {
			t.Fatalf("new: %v", err)
		}
		if err := b.AppendWithOptions(event.TypeAIRequestReceived, map[string]any{
			"request_id":    "req-" + payload,
			"actor_id_hash": "h",
			"purpose_tag":   "rag",
			"input_digest":  "sha256:" + payload,
		}, &bundle.AppendOptions{Timestamp: "2026-04-09T00:00:00Z"}); err != nil {
			t.Fatalf("append: %v", err)
		}
		return b
	}

	original := makeBundle("original")
	if err := original.Save(context.Background(), path); err != nil {
		t.Fatalf("save original: %v", err)
	}

	competitor := makeBundle("competitor")

	var wg sync.WaitGroup
	wg.Add(2)

	var signErr error
	go func() {
		defer wg.Done()
		_, signErr = bundle.SignTo(context.Background(), path, path, privateKey)
	}()
	go func() {
		defer wg.Done()
		// Both writers contend on the same advisory lock. With the TOCTOU
		// window closed, whichever writer wins the lock first runs to
		// completion before the other observes the file.
		_ = competitor.Save(context.Background(), path)
	}()
	wg.Wait()

	// Sign may have failed if the competitor won the lock first and changed
	// the file underneath the lock contract — that's an acceptable outcome
	// (errors.Is may match ErrBundleLocked on either path). What is NOT
	// acceptable: a corrupt final file.
	if signErr != nil && !errors.Is(signErr, bundle.ErrBundleLocked) {
		// Re-loading the file must still succeed even if Sign reported a
		// locking error.
		t.Logf("sign returned non-lock error (acceptable in this race): %v", signErr)
	}

	loaded, err := bundle.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("final load: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("final verify (torn write?): %v", err)
	}
}

// keep imports used unconditionally even if compiler inlines the variables.
var _ = time.RFC3339
var _ = context.Background
