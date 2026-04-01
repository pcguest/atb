package verify

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestClassifyAnchor_Absent(t *testing.T) {
	if got := ClassifyAnchor(bundle.New(), "bundle.atb"); got != AnchorAbsent {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorAbsent)
	}
}

func TestClassifyAnchor_PresentBadData(t *testing.T) {
	fixture := readAnchorTSRFixture(t)
	b := bundle.New()
	appendVerifyRecord(t, b, event.TypeBundleAnchor, mustMarshalAnchorEventData(t, fixture), "2026-03-28T03:04:05Z")

	if got := ClassifyAnchor(b, ""); got != AnchorPresentBadData {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorPresentBadData)
	}
}

func TestClassifyAnchor_DigestMismatch(t *testing.T) {
	fixture := readAnchorTSRFixture(t)
	b := bundle.New()
	appendVerifyRecord(t, b, event.TypeBundleAnchor, mustMarshalAnchorEventData(t, fixture), "2026-03-28T03:04:05Z")

	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	if got := ClassifyAnchor(b, path); got != AnchorPresentBadData {
		t.Fatalf("ClassifyAnchor() = %v, want %v", got, AnchorPresentBadData)
	}
}

func TestClassifyAnchor_Verified(t *testing.T) {
	t.Skip("ClassifyAnchor uses anchor.VerifyToken with system roots only; no system-trusted offline TSA fixture exists in testdata")
}

func readAnchorTSRFixture(t testing.TB) []byte {
	t.Helper()

	fixture, err := os.ReadFile(filepath.Join("..", "anchor", "testdata", "sample.tsr"))
	if err != nil {
		t.Fatalf("read anchor fixture: %v", err)
	}
	return fixture
}

func mustMarshalAnchorEventData(t testing.TB, tsrDER []byte) string {
	t.Helper()

	tsrHash := sha256.Sum256(tsrDER)
	payload := map[string]any{
		"bundle_hash":    "bundle-hash",
		"tsr_hash":       hex.EncodeToString(tsrHash[:]),
		"tsr_der":        base64.StdEncoding.EncodeToString(tsrDER),
		"certified_time": "2026-03-28T03:04:05Z",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal anchor payload: %v", err)
	}
	return string(raw)
}
