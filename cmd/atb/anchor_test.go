package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
)

func TestRunAnchorWritesTSRAndAppendsAnchorEvent(t *testing.T) {
	fixture := readAnchorFixture(t)
	bundlePath, originalHash := writeAnchorTestBundle(t)
	stubTSATransport(t, fixture)

	result, err := runAnchor(anchorConfig{
		BundlePath: bundlePath,
		TSAURL:     "http://tsa.example.test",
	})
	if err != nil {
		t.Fatalf("runAnchor: %v", err)
	}
	t.Logf("anchor_event_data=%s", result.EventData)

	if result.TokenPath != bundlePath+".tsr" {
		t.Fatalf("unexpected token path: got %q", result.TokenPath)
	}

	tokenBytes, err := os.ReadFile(result.TokenPath)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if !bytes.Equal(tokenBytes, fixture) {
		t.Fatalf("unexpected token bytes")
	}

	loaded, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load anchored bundle: %v", err)
	}
	if got := loaded.Records[len(loaded.Records)-1].Event.Type; got != bundle.AnchorEventType {
		t.Fatalf("unexpected final event type: got %q", got)
	}

	anchorIndex, data, found, err := latestAnchorEventData(loaded)
	if err != nil {
		t.Fatalf("latestAnchorEventData: %v", err)
	}
	if !found {
		t.Fatalf("expected anchor event to be present")
	}
	if anchorIndex != len(loaded.Records)-1 {
		t.Fatalf("expected latest anchor event to be the final record")
	}

	if data.TSAURL != "http://tsa.example.test" {
		t.Fatalf("unexpected tsa_url: got %q", data.TSAURL)
	}
	if data.BundleHash != hex.EncodeToString(originalHash) {
		t.Fatalf("unexpected bundle_hash: got %q", data.BundleHash)
	}
	tsrHash := sha256.Sum256(fixture)
	if data.TSRHash != hex.EncodeToString(tsrHash[:]) {
		t.Fatalf("unexpected tsr_hash: got %q", data.TSRHash)
	}
	if data.CertifiedTime != "2026-03-28T03:04:05Z" {
		t.Fatalf("unexpected certified_time: got %q", data.CertifiedTime)
	}
}

func TestVerifyWithAnchorPassesOnAnchoredBundle(t *testing.T) {
	fixture := readAnchorFixture(t)
	bundlePath, _ := writeAnchorTestBundle(t)
	stubTSATransport(t, fixture)

	if _, err := runAnchor(anchorConfig{
		BundlePath: bundlePath,
		TSAURL:     "http://tsa.example.test",
	}); err != nil {
		t.Fatalf("runAnchor: %v", err)
	}

	loaded, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load anchored bundle: %v", err)
	}
	if err := loaded.Verify(); err != nil {
		t.Fatalf("verify anchored bundle chain: %v", err)
	}

	var out bytes.Buffer
	if err := verifyBundleAnchor(bundlePath, loaded, &out); err != nil {
		t.Fatalf("verifyBundleAnchor: %v", err)
	}
	if !strings.Contains(out.String(), "Anchor verified. Certified: 2026-03-28T03:04:05Z") {
		t.Fatalf("expected certified time output, got %q", out.String())
	}
}

func TestVerifyWithAnchorWarnsWhenTokenAbsent(t *testing.T) {
	fixture := readAnchorFixture(t)
	bundlePath, _ := writeAnchorTestBundle(t)
	stubTSATransport(t, fixture)

	result, err := runAnchor(anchorConfig{
		BundlePath: bundlePath,
		TSAURL:     "http://tsa.example.test",
	})
	if err != nil {
		t.Fatalf("runAnchor: %v", err)
	}
	if err := os.Remove(result.TokenPath); err != nil {
		t.Fatalf("remove token file: %v", err)
	}

	loaded, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load anchored bundle: %v", err)
	}

	var out bytes.Buffer
	if err := verifyBundleAnchor(bundlePath, loaded, &out); err != nil {
		t.Fatalf("verifyBundleAnchor: %v", err)
	}
	want := "No anchor token found at " + result.TokenPath + " — skipping anchor verification"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("expected missing token warning, got %q", out.String())
	}
}

func readAnchorFixture(t *testing.T) []byte {
	t.Helper()

	fixture, err := os.ReadFile(filepath.Join("..", "..", "internal", "anchor", "testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read anchor fixture: %v", err)
	}
	return fixture
}

func writeAnchorTestBundle(t *testing.T) (string, []byte) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "run.atb", bundle.BundleFile)
	b := bundle.New()
	appendTestBundleEvent(t, b, "dev.session", map[string]any{"ok": true})
	if err := b.Save(path); err != nil {
		t.Fatalf("save test bundle: %v", err)
	}
	hash, err := anchorpkg.HashBundle(path)
	if err != nil {
		t.Fatalf("hash test bundle: %v", err)
	}
	return path, hash
}

func stubTSATransport(t *testing.T, fixture []byte) {
	t.Helper()

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: got %s want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/timestamp-query" {
			t.Fatalf("unexpected content type: got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if len(body) == 0 {
			t.Fatalf("expected non-empty TSA request body")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     make(http.Header),
		}, nil
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
