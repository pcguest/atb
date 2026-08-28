// SPDX-License-Identifier: MIT
package hash_test

// TestHashGoldenCorpus pins the byte-for-byte output of canonicalisation
// and the SHA-256 hash chain against a committed corpus that is mirrored
// in the Python SDK (sdk/python/tests/test_hash_golden.py).
//
// A failure here means an Event field, struct tag, canonicalisation rule,
// or float serialisation has changed in a way that rewrites bundle hashes —
// a BREAKING SCHEMA CHANGE. Bump ManifestVersion (docs/specification/bundle-v1.md §8),
// add a CHANGELOG entry, regenerate testdata/golden_corpus.json, and update
// the Python mirror in lockstep.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/hash"
)

const goldenCorpusPath = "testdata/golden_corpus.json"

type goldenCorpusEntry struct {
	Description  string     `json:"description"`
	Event        hash.Event `json:"event"`
	CanonicalHex string     `json:"canonical_hex"`
	HashHex      string     `json:"hash_hex"`
}

func TestHashGoldenCorpus(t *testing.T) {
	raw, err := os.ReadFile(goldenCorpusPath)
	if err != nil {
		t.Fatalf("read golden corpus: %v", err)
	}
	var entries []goldenCorpusEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("parse golden corpus: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("golden corpus is empty")
	}

	for _, entry := range entries {
		t.Run(entry.Description, func(t *testing.T) {
			canonical, err := canonicalize.Marshal(entry.Event)
			if err != nil {
				t.Fatalf("canonicalize: %v", err)
			}
			gotCanonical := hex.EncodeToString(canonical)
			if gotCanonical != entry.CanonicalHex {
				t.Errorf("canonical bytes changed — this is a breaking schema change, bump ManifestVersion and update the golden corpus\n  want: %s\n  got:  %s", entry.CanonicalHex, gotCanonical)
			}

			h := sha256.New()
			h.Write([]byte(entry.Event.PrevHash))
			h.Write(canonical)
			gotHash := hex.EncodeToString(h.Sum(nil))
			if gotHash != entry.HashHex {
				t.Errorf("hash changed — this is a breaking schema change, bump ManifestVersion and update the golden corpus\n  want: %s\n  got:  %s", entry.HashHex, gotHash)
			}

			computed, err := hash.Compute(entry.Event)
			if err != nil {
				t.Fatalf("hash.Compute: %v", err)
			}
			if computed != entry.HashHex {
				t.Errorf("hash.Compute drift\n  want: %s\n  got:  %s", entry.HashHex, computed)
			}
		})
	}
}
