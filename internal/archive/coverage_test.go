// SPDX-License-Identifier: MIT
package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/hash"
)

func TestLedgerErrorContracts(t *testing.T) {
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.ndjson")
	if err := os.WriteFile(malformed, []byte("\n{\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(malformed); err == nil || !strings.Contains(err.Error(), "parse archive ledger") {
		t.Fatalf("malformed ledger error=%v", err)
	}
	if _, err := LoadOrEmpty(malformed); err == nil {
		t.Fatal("LoadOrEmpty swallowed malformed ledger")
	}

	if _, err := Verify([]Entry{{Sequence: 2, PrevHash: hash.GenesisHash}}); err == nil ||
		!strings.Contains(err.Error(), "has seq") {
		t.Fatalf("sequence error=%v", err)
	}
	if _, err := Verify([]Entry{{Sequence: 1, PrevHash: "wrong"}}); err == nil ||
		!strings.Contains(err.Error(), "prev_hash") {
		t.Fatalf("prev-hash error=%v", err)
	}
	if _, err := NextEntry([]Entry{{Sequence: 2, PrevHash: hash.GenesisHash}}, "", "", "", "", ""); err == nil {
		t.Fatal("NextEntry accepted invalid chain")
	}

	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, []byte("block"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Append(filepath.Join(blocker, LedgerFile), Entry{})
	if err == nil || !strings.Contains(err.Error(), "mkdir") {
		t.Fatalf("append mkdir error=%v", err)
	}
}
