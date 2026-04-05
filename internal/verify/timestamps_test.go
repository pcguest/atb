package verify

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func TestValidateTimestamps_IgnoresMissingAndAcceptsNonDecreasing(t *testing.T) {
	records := []bundle.Record{
		{Event: hash.Event{Sequence: 0, Type: "dev.session", Timestamp: ""}},
		{Event: hash.Event{Sequence: 1, Type: "ai.request.received", Timestamp: "2026-03-27T12:00:00Z"}},
		{Event: hash.Event{Sequence: 2, Type: "ai.policy.decision", Timestamp: "2026-03-27T12:00:00Z"}},
		{Event: hash.Event{Sequence: 3, Type: "ai.action.executed", Timestamp: "2026-03-27T12:01:00Z"}},
	}

	if violations := ValidateTimestamps(records); len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateTimestamps_ReportsFormatAndOrderViolations(t *testing.T) {
	records := []bundle.Record{
		{Event: hash.Event{Sequence: 0, Type: "ai.request.received", Timestamp: "2026-03-27T12:00:00Z"}},
		{Event: hash.Event{Sequence: 1, Type: "ai.policy.decision", Timestamp: "bad-time"}},
		{Event: hash.Event{Sequence: 2, Type: "ai.action.executed", Timestamp: "2026-03-27T11:59:00Z"}},
	}

	violations := ValidateTimestamps(records)
	if got, want := len(violations), 2; got != want {
		t.Fatalf("unexpected violation count: got %d want %d (%v)", got, want, violations)
	}
	if violations[0] != `timestamp validation: seq 1 (ai.policy.decision) has invalid RFC 3339 timestamp "bad-time"` {
		t.Fatalf("unexpected format violation: %q", violations[0])
	}
	if violations[1] != `timestamp validation: seq 2 (ai.action.executed) timestamp "2026-03-27T11:59:00Z" is earlier than the preceding timestamp "2026-03-27T12:00:00Z"` {
		t.Fatalf("unexpected order violation: %q", violations[1])
	}
}
