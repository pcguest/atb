// SPDX-License-Identifier: MIT
package bundle_test

import (
	"crypto/sha256"
	"encoding/hex"
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

// TestAdversarialTamperVectors covers the full set of named tamper vectors
// from the security review. Each subtest mutates an in-memory bundle and
// asserts that Verify rejects it. Cases that cannot be detected without an
// out-of-band terminal commitment (e.g. tail truncation of an unsigned
// bundle) are documented inline.
func TestAdversarialTamperVectors(t *testing.T) {
	t.Run("tamper_record_zero_manifest", func(t *testing.T) {
		b := newAdversarialTestBundle(t, 4)
		b.Records[0].Event.Data = "tampered manifest payload"
		assertVerifyError(t, b)
	})

	t.Run("tamper_last_record", func(t *testing.T) {
		b := newAdversarialTestBundle(t, 4)
		last := len(b.Records) - 1
		b.Records[last].Event.Data = map[string]any{"step": -1}
		assertVerifyError(t, b)
	})

	t.Run("tamper_middle_record", func(t *testing.T) {
		b := newAdversarialTestBundle(t, 5)
		mid := len(b.Records) / 2
		b.Records[mid].Event.Type = "ai.tool.middle_tampered"
		assertVerifyError(t, b)
	})

	t.Run("delete_record_zero_manifest", func(t *testing.T) {
		// Removing the manifest leaves event #1 with prev_hash=genesis (valid)
		// but the record at index 0 is now a non-manifest event whose own
		// stored hash still chains correctly. Verify must still reject this:
		// either via manifest-presence checks or via sequence drift, since
		// the surviving records still claim seq>=1 starting at position 0.
		b := newAdversarialTestBundle(t, 4)
		b.Records = b.Records[1:]
		assertVerifyError(t, b)
	})

	t.Run("delete_middle_record", func(t *testing.T) {
		b := newAdversarialTestBundle(t, 5)
		b.Records = append(b.Records[:2], b.Records[3:]...)
		assertVerifyError(t, b)
	})

	t.Run("reorder_adjacent_records", func(t *testing.T) {
		b := newAdversarialTestBundle(t, 4)
		b.Records[2], b.Records[3] = b.Records[3], b.Records[2]
		assertVerifyError(t, b)
	})

	t.Run("insert_record_mid_chain", func(t *testing.T) {
		// Build a separate valid bundle and splice one of its records into
		// the middle of the target. The injected record carries a valid
		// hash for its original position but breaks the chain wherever it
		// is inserted, since both prev_hash and seq diverge.
		victim := newAdversarialTestBundle(t, 4)
		donor := newAdversarialTestBundle(t, 3)
		injected := donor.Records[2]
		victim.Records = append(victim.Records[:2], append([]bundle.Record{injected}, victim.Records[2:]...)...)
		assertVerifyError(t, victim)
	})

	t.Run("duplicate_last_record", func(t *testing.T) {
		b := newAdversarialTestBundle(t, 4)
		b.Records = append(b.Records, b.Records[len(b.Records)-1])
		assertVerifyError(t, b)
	})

	t.Run("truncate_half", func(t *testing.T) {
		// A plain unsigned bundle's prefix is internally consistent: any
		// suffix-truncation produces a shorter but valid hash chain.
		// Detection requires a terminal commitment (signature or anchor),
		// covered in internal/sign and internal/anchor test suites.
		t.Skip("unsigned bundle truncation preserves a valid prefix; requires terminal commitment")
	})

	t.Run("relink_prev_hash_to_sha256_of_wrong_content", func(t *testing.T) {
		// Replace prev_hash with the SHA-256 of arbitrary content. The
		// substituted value is well-formed (64 hex chars, valid digest)
		// but does not match the actual previous record's hash, so the
		// chain check fails.
		b := newAdversarialTestBundle(t, 4)
		sum := sha256.Sum256([]byte("attacker-chosen prefix"))
		b.Records[2].Event.PrevHash = hex.EncodeToString(sum[:])
		assertVerifyError(t, b)
	})

	t.Run("replace_data_field_only", func(t *testing.T) {
		// Confirm the per-event hash covers the data payload: mutating
		// only Event.Data while leaving every other field (including the
		// stored Hash) untouched must be detected.
		b := newAdversarialTestBundle(t, 4)
		b.Records[2].Event.Data = map[string]any{"step": "exfiltrated"}
		assertVerifyError(t, b)
	})
}
