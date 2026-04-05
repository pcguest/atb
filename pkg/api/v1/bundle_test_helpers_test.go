package apiv1

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func testBundleTimestamp() string {
	return time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func testBundleID() string {
	return strings.Repeat("a", 32)
}

func newTestBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	return b
}

func appendTestBundleEvent(t testing.TB, b *bundle.Bundle, eventType string, data interface{}) {
	t.Helper()
	appendTestBundleEventAt(t, b, eventType, data, testBundleTimestamp())
}

func appendTestBundleEventAt(
	t testing.TB,
	b *bundle.Bundle,
	eventType string,
	data interface{},
	timestamp string,
) {
	t.Helper()

	stabiliseTestBundleManifest(t, b)

	if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
}

func stabiliseTestBundleManifest(t testing.TB, b *bundle.Bundle) {
	t.Helper()

	manifest := b.Manifest()
	if manifest == nil || manifest.BundleID == testBundleID() {
		return
	}

	payload, err := json.Marshal(bundle.ManifestData{
		Version:   bundle.ManifestVersion,
		CreatedAt: testBundleTimestamp(),
		BundleID:  testBundleID(),
	})
	if err != nil {
		t.Fatalf("marshal fixed manifest payload: %v", err)
	}

	b.Records[0].Event.Data = string(payload)
	b.Records[0].Event.Timestamp = testBundleTimestamp()
	rehashTestBundle(t, b)
}

func rehashTestBundle(t testing.TB, b *bundle.Bundle) {
	t.Helper()

	prev := hash.GenesisHash
	hasManifest := len(b.Records) > 0 && b.Records[0].Event.Type == bundle.ManifestEventType
	for i := range b.Records {
		b.Records[i].Event.PrevHash = prev
		switch {
		case hasManifest && i == 0:
			b.Records[i].Event.Sequence = 0
		case hasManifest:
			b.Records[i].Event.Sequence = i
		default:
			b.Records[i].Event.Sequence = i + 1
		}

		computed, err := hash.Compute(b.Records[i].Event)
		if err != nil {
			t.Fatalf("rehash record %d: %v", i, err)
		}
		b.Records[i].Hash = computed
		prev = computed
	}
}
