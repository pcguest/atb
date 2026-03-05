package archive

import (
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/hash"
)

func TestLedgerHashChainUniqueness(t *testing.T) {
	entry := Entry{
		Sequence:   1,
		PrevHash:   hash.GenesisHash,
		ArchivedAt: "2026-03-05T10:00:00Z",
		Source:     "run.atb/bundle.atb",
		Dest:       "archive.atb/2026/03/05/run.atb/bundle.atb",
		SHA256:     "abc123",
		HeadHash:   "def456",
	}
	otherSeq := entry
	otherSeq.Sequence = 2

	h1, err := ComputeHash(entry)
	if err != nil {
		t.Fatalf("compute hash for first entry: %v", err)
	}
	h2, err := ComputeHash(otherSeq)
	if err != nil {
		t.Fatalf("compute hash for second entry: %v", err)
	}
	if h1 == h2 {
		t.Fatalf("expected different hashes for different seq values")
	}
}

func TestLedgerRoundTripAndVerify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.atb", LedgerFile)

	entries, err := LoadOrEmpty(path)
	if err != nil {
		t.Fatalf("load empty ledger: %v", err)
	}

	e1, err := NextEntry(entries, "2026-03-05T10:00:00Z", "run.atb/one.atb", "archive.atb/2026/03/05/run.atb/one.atb", "sha1", "head1")
	if err != nil {
		t.Fatalf("next entry 1: %v", err)
	}
	if err := Append(path, e1); err != nil {
		t.Fatalf("append entry 1: %v", err)
	}

	entries, err = Load(path)
	if err != nil {
		t.Fatalf("load ledger after append: %v", err)
	}
	e2, err := NextEntry(entries, "2026-03-05T11:00:00Z", "run.atb/two.atb", "archive.atb/2026/03/05/run.atb/two.atb", "sha2", "head2")
	if err != nil {
		t.Fatalf("next entry 2: %v", err)
	}
	if err := Append(path, e2); err != nil {
		t.Fatalf("append entry 2: %v", err)
	}

	entries, err = Load(path)
	if err != nil {
		t.Fatalf("reload ledger: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected entries length: got %d want 2", len(entries))
	}
	head, err := Verify(entries)
	if err != nil {
		t.Fatalf("verify ledger: %v", err)
	}
	if head == hash.GenesisHash {
		t.Fatalf("expected non-genesis head hash for non-empty ledger")
	}
}
