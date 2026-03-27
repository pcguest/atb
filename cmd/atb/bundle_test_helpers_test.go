package main

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
	appendTestBundleEventWithOptions(t, b, eventType, data, nil)
}

func appendTestBundleEventWithOptions(
	t testing.TB,
	b *bundle.Bundle,
	eventType string,
	data interface{},
	opts *bundle.AppendOptions,
) {
	t.Helper()

	appendOpts := bundle.AppendOptions{
		Timestamp: testBundleTimestamp(),
	}
	if opts != nil {
		appendOpts = *opts
		if appendOpts.Timestamp == "" {
			appendOpts.Timestamp = testBundleTimestamp()
		}
	}

	if err := b.AppendWithOptions(eventType, data, &appendOpts); err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
}
