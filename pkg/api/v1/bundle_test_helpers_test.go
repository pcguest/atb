package apiv1

import (
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

func testBundleTimestamp() string {
	return time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
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

	if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{Timestamp: timestamp}); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
}
