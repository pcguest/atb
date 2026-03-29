package trust

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestVerifyAnchors_NoAnchors(t *testing.T) {
	b := bundle.New()
	if err := b.Append(event.TypeDevSession, map[string]any{"event_id": "evt-1"}); err != nil {
		t.Fatalf("append dev session: %v", err)
	}

	results := VerifyAnchors(b)
	if len(results) != 0 {
		t.Fatalf("expected no anchor results, got %d", len(results))
	}
}

func TestVerifyAnchors_MalformedData(t *testing.T) {
	testVerifyAnchorErrorCase(t, anchorErrorCase{
		name: "malformed data",
		data: "not-valid-json",
	})
}

func TestVerifyAnchors_EmptyTSPBytes(t *testing.T) {
	payload := marshalAnchorFixturePayload(t, "", "")
	testVerifyAnchorErrorCase(t, anchorErrorCase{
		name: "empty embedded tsr bytes",
		data: payload,
	})
}

func TestVerifyAnchors_FakeTSPBytes(t *testing.T) {
	payload := marshalAnchorFixturePayload(t, "2e68c5301eb2fae7449b7f8db6f3b80f03ac1ff221bd13a96f989e65ca2ff552", base64.StdEncoding.EncodeToString([]byte("not-a-valid-tsp-response")))
	testVerifyAnchorErrorCase(t, anchorErrorCase{
		name: "fake embedded tsr bytes",
		data: payload,
	})
}

type anchorErrorCase struct {
	name string
	data string
}

func testVerifyAnchorErrorCase(t *testing.T, tc anchorErrorCase) {
	t.Helper()

	b := bundle.New()
	if err := b.Append(event.TypeDevSession, map[string]any{"event_id": tc.name}); err != nil {
		t.Fatalf("append dev session: %v", err)
	}
	b.Records[len(b.Records)-1].Event.Type = event.TypeBundleAnchor
	b.Records[len(b.Records)-1].Event.Data = tc.data

	results := VerifyAnchors(b)
	if len(results) != 1 {
		t.Fatalf("expected 1 anchor result, got %d", len(results))
	}
	if results[0].Error == "" {
		t.Fatalf("expected non-empty error for %s", tc.name)
	}
	if results[0].MessageImprintMatch {
		t.Fatalf("expected MessageImprintMatch=false for %s", tc.name)
	}
}

func marshalAnchorFixturePayload(t testing.TB, tsrHash string, tsrDER string) string {
	t.Helper()

	payload, err := json.Marshal(map[string]string{
		"tsa_url":        "http://tsa.example.test",
		"bundle_hash":    "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f8a3b2d4c1f97cf0a2",
		"tsr_hash":       tsrHash,
		"tsr_der":        tsrDER,
		"certified_time": "2026-03-29T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal anchor payload: %v", err)
	}
	return string(payload)
}
