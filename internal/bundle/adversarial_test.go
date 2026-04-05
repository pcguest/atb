package bundle_test

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func newAdversarialTestBundle(t testing.TB, recordCount int) *bundle.Bundle {
	t.Helper()

	if recordCount < 1 {
		t.Fatalf("recordCount must be at least 1, got %d", recordCount)
	}

	b := newTestBundle(t)
	for step := 1; len(b.Records) < recordCount; step++ {
		if err := b.Append("ai.tool.exec", map[string]any{"step": step}); err != nil {
			t.Fatalf("append record %d: %v", step, err)
		}
	}
	return b
}

func assertVerifyError(t testing.TB, b *bundle.Bundle) {
	t.Helper()

	if err := b.Verify(); err == nil {
		t.Fatal("expected Verify to return an error")
	}
}

// TestTamperEventType covers changing an event type after the bundle is created.
func TestTamperEventType(t *testing.T) {
	b := newAdversarialTestBundle(t, 3)

	b.Records[2].Event.Type = "ai.tool.tampered"

	assertVerifyError(t, b)
}

// TestTamperEventData covers changing an event payload after the bundle is created.
func TestTamperEventData(t *testing.T) {
	b := newAdversarialTestBundle(t, 3)

	b.Records[2].Event.Data = map[string]any{"step": 999}

	assertVerifyError(t, b)
}

// TestTamperPrevHash covers changing the stored prev_hash link for a record.
func TestTamperPrevHash(t *testing.T) {
	b := newAdversarialTestBundle(t, 3)

	b.Records[2].Event.PrevHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	assertVerifyError(t, b)
}

// TestTamperHashField covers changing the stored hash field for a record.
func TestTamperHashField(t *testing.T) {
	b := newAdversarialTestBundle(t, 3)

	b.Records[2].Hash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	assertVerifyError(t, b)
}

// TestDeleteMiddleRecord covers deleting an interior record from an otherwise valid bundle.
func TestDeleteMiddleRecord(t *testing.T) {
	b := newAdversarialTestBundle(t, 4)

	b.Records = append(b.Records[:2], b.Records[3:]...)

	assertVerifyError(t, b)
}

// TestTruncateBundle covers truncating the tail of a bundle after records were appended.
func TestTruncateBundle(t *testing.T) {
	// A plain unsigned bundle prefix remains internally consistent after tail
	// truncation, so Verify cannot reject this without an explicit terminal
	// commitment such as signing or anchoring.
	t.Skip("unsigned bundle tail truncation preserves a valid prefix; requires terminal commitment")
}

// TestDuplicateRecord covers inserting a duplicate record into the bundle sequence.
func TestDuplicateRecord(t *testing.T) {
	b := newAdversarialTestBundle(t, 3)

	dup := b.Records[2]
	b.Records = append(b.Records, dup)

	assertVerifyError(t, b)
}

// TestSwapRecords covers swapping two adjacent records in a valid bundle.
func TestSwapRecords(t *testing.T) {
	b := newAdversarialTestBundle(t, 4)

	b.Records[2], b.Records[3] = b.Records[3], b.Records[2]

	assertVerifyError(t, b)
}

// TestTamperSequenceNumber covers changing a record sequence number after hashing.
func TestTamperSequenceNumber(t *testing.T) {
	b := newAdversarialTestBundle(t, 3)

	b.Records[2].Event.Sequence = 99

	assertVerifyError(t, b)
}

// TestAppendAfterSign covers appending to a bundle after bundle signing integration.
func TestAppendAfterSign(t *testing.T) {
	// Signing lives in the CLI integration path, so the end-to-end append-after-sign
	// behavior is exercised in cmd/atb/sign_test.go rather than internal/bundle.
	t.Skip("bundle signing integration: tested in cmd/atb/sign_test.go")
}
