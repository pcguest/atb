package anchor

import (
	"bytes"
	"crypto/sha256"
	"encoding/asn1"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTSReqEncodesSHA256OID(t *testing.T) {
	req, err := buildTSReq(bytes.Repeat([]byte{0x42}, sha256.Size))
	if err != nil {
		t.Fatalf("buildTSReq: %v", err)
	}

	var raw asn1.RawValue
	rest, err := asn1.Unmarshal(req, &raw)
	if err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if len(rest) != 0 {
		t.Fatalf("expected no trailing bytes, got %d", len(rest))
	}

	sha256OIDDER := []byte{0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01}
	if !bytes.Contains(req, sha256OIDDER) {
		t.Fatalf("expected request to contain SHA-256 OID DER %x", sha256OIDDER)
	}
}

func TestHashBundle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	content := []byte("{\"event\":{\"seq\":0},\"hash\":\"abc\"}\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write temp bundle: %v", err)
	}

	got, err := HashBundle(path)
	if err != nil {
		t.Fatalf("HashBundle: %v", err)
	}

	want := sha256.Sum256(content)
	if !bytes.Equal(got, want[:]) {
		t.Fatalf("unexpected bundle hash: got %x want %x", got, want)
	}
}

func TestParseGenTime(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got, err := ParseGenTime(fixture)
	if err != nil {
		t.Fatalf("ParseGenTime: %v", err)
	}
	if got != "2026-03-28T03:04:05Z" {
		t.Fatalf("unexpected genTime: got %q", got)
	}
}

func TestRequestPostsTimestampQuery(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

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
			t.Fatalf("expected non-empty request body")
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(fixture)),
			Header:     make(http.Header),
		}, nil
	})

	got, err := Request("http://tsa.example.test", bytes.Repeat([]byte{0x24}, sha256.Size))
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if !bytes.Equal(got, fixture) {
		t.Fatalf("unexpected response bytes")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
