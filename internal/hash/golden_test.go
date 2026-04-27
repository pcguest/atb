// SPDX-License-Identifier: MIT
package hash_test

// TestCanonicalHashGolden pins the byte-for-byte output of ATB's
// RFC 8785 + SHA-256 canonicalisation pipeline to a committed corpus.
//
// Any change that causes this test to fail is a BREAKING SCHEMA CHANGE.
// It requires a manifest version bump (docs/spec-v1.0.md §8), a CHANGELOG
// entry, and regeneration of the golden file with the new expected values.
// The same corpus is mirrored in the Python and TypeScript SDKs; do not
// regenerate this file without updating those.
//
// To regenerate after a deliberate, vetted format change:
//
//	go test ./internal/hash/... -run TestCanonicalHashGolden -update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/hash"
)

var updateGolden = flag.Bool("update", false, "regenerate internal/hash/testdata/golden.json from the live canonicalisation pipeline")

const goldenPath = "testdata/golden.json"

// goldenCase is one entry in the committed corpus. The Event field is held
// as RawMessage so the test exercises the same JSON → canonical path that a
// third-party verifier would walk.
type goldenCase struct {
	Description               string          `json:"description"`
	Event                     json.RawMessage `json:"event"`
	PrevHash                  string          `json:"prev_hash"`
	ExpectedCanonicalBytesHex string          `json:"expected_canonical_bytes_hex"`
	ExpectedHashHex           string          `json:"expected_hash_hex"`
	// ChainPrevFromIndex, when non-nil, signals that this case's PrevHash
	// is the ExpectedHashHex of cases[*ChainPrevFromIndex]. -update enforces
	// this; normal runs assert it.
	ChainPrevFromIndex *int `json:"chain_prev_from_index,omitempty"`
}

// genesisHash duplicates hash.GenesisHash to keep this file self-contained.
const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// seedCases defines the inputs of the corpus. Expected outputs are written
// to disk by -update and read by every other run.
func seedCases() []goldenCase {
	chainPrevFrom := func(i int) *int { return &i }

	return []goldenCase{
		{
			Description: "minimal: required fields only, data is a string",
			Event: json.RawMessage(`{
				"seq": 1,
				"prev_hash": "` + genesisHash + `",
				"type": "test.minimal",
				"hash_algo": "sha256",
				"data": "hello"
			}`),
			PrevHash: genesisHash,
		},
		{
			Description: "all optional fields present",
			Event: json.RawMessage(`{
				"seq": 5,
				"prev_hash": "1111111111111111111111111111111111111111111111111111111111111111",
				"type": "test.all_fields",
				"hash_algo": "sha256",
				"data": "payload",
				"actor_id": "alice",
				"org_id": "acme",
				"workspace_id": "ws-1",
				"timestamp": "2026-01-02T03:04:05Z",
				"trace_id": "0123456789abcdef0123456789abcdef",
				"span_id": "0123456789abcdef",
				"parent_span_id": "fedcba9876543210"
			}`),
			PrevHash: "1111111111111111111111111111111111111111111111111111111111111111",
		},
		{
			Description: "data is a JSON object",
			Event: json.RawMessage(`{
				"seq": 2,
				"prev_hash": "` + genesisHash + `",
				"type": "test.data_object",
				"hash_algo": "sha256",
				"data": {"key": "value", "n": 42, "nested": {"a": [1, 2, 3]}}
			}`),
			PrevHash: genesisHash,
		},
		{
			Description: "data is null",
			Event: json.RawMessage(`{
				"seq": 3,
				"prev_hash": "` + genesisHash + `",
				"type": "test.data_null",
				"hash_algo": "sha256",
				"data": null
			}`),
			PrevHash: genesisHash,
		},
		{
			Description: "data contains a float that exercises the v1.1.2 exponential rule (large)",
			Event: json.RawMessage(`{
				"seq": 4,
				"prev_hash": "` + genesisHash + `",
				"type": "test.data_float_large",
				"hash_algo": "sha256",
				"data": {"n": 1e21}
			}`),
			PrevHash: genesisHash,
		},
		{
			Description: "data contains a small float that exercises the v1.1.2 exponential rule (sub-1e-6)",
			Event: json.RawMessage(`{
				"seq": 5,
				"prev_hash": "` + genesisHash + `",
				"type": "test.data_float_small",
				"hash_algo": "sha256",
				"data": {"n": 1.5e-7}
			}`),
			PrevHash: genesisHash,
		},
		{
			Description: "unicode in data (Japanese plus emoji)",
			Event: json.RawMessage(`{
				"seq": 6,
				"prev_hash": "` + genesisHash + `",
				"type": "test.unicode",
				"hash_algo": "sha256",
				"data": "こんにちは 🎉"
			}`),
			PrevHash: genesisHash,
		},
		{
			Description: "genesis prev_hash sentinel as an explicit case",
			Event: json.RawMessage(`{
				"seq": 1,
				"prev_hash": "` + genesisHash + `",
				"type": "test.genesis",
				"hash_algo": "sha256",
				"data": {}
			}`),
			PrevHash: genesisHash,
		},
		{
			Description: "consecutive event #1 (chain head)",
			Event: json.RawMessage(`{
				"seq": 1,
				"prev_hash": "` + genesisHash + `",
				"type": "test.chain_head",
				"hash_algo": "sha256",
				"data": "first"
			}`),
			PrevHash: genesisHash,
		},
		{
			Description:        "consecutive event #2 (prev_hash is event #1's expected hash)",
			Event:              nil,              // filled in below; depends on previous expected hash
			ChainPrevFromIndex: chainPrevFrom(8), // index of "chain head"
		},
	}
}

// renderEventForChainedCase fills in the JSON for the chained case using the
// resolved PrevHash. Kept separate so the seed table stays declarative.
func renderEventForChainedCase(prev string) json.RawMessage {
	return json.RawMessage(`{
		"seq": 2,
		"prev_hash": "` + prev + `",
		"type": "test.chain_next",
		"hash_algo": "sha256",
		"data": "second"
	}`)
}

func computeExpected(t *testing.T, eventJSON json.RawMessage, prevHash string) (canonicalHex, hashHex string) {
	t.Helper()
	canonical, err := canonicalize.MarshalRaw(eventJSON)
	if err != nil {
		t.Fatalf("canonicalize event: %v", err)
	}
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write(canonical)
	return hex.EncodeToString(canonical), hex.EncodeToString(h.Sum(nil))
}

func TestCanonicalHashGolden(t *testing.T) {
	if *updateGolden {
		regenerateGolden(t)
		return
	}

	abs, _ := filepath.Abs(goldenPath)
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf(
			"golden corpus missing or unreadable at %s: %v\n"+
				"To bootstrap or regenerate after a vetted format change, run:\n"+
				"  go test ./internal/hash/... -run TestCanonicalHashGolden -update",
			abs, err,
		)
	}

	var cases []goldenCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse golden corpus: %v", err)
	}

	seed := seedCases()
	if len(cases) != len(seed) {
		t.Fatalf("golden case count (%d) != seed count (%d); regenerate with -update", len(cases), len(seed))
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case_%02d_%s", i, c.Description), func(t *testing.T) {
			// 1. Cross-check chain dependency, if declared.
			if seed[i].ChainPrevFromIndex != nil {
				want := cases[*seed[i].ChainPrevFromIndex].ExpectedHashHex
				if c.PrevHash != want {
					t.Fatalf("chained prev_hash drift: case %d prev_hash=%s, expected case %d expected_hash_hex=%s",
						i, c.PrevHash, *seed[i].ChainPrevFromIndex, want)
				}
			}

			// 2. Recompute canonical bytes and hash from the recorded event JSON.
			gotCanonicalHex, gotHashHex := computeExpected(t, c.Event, c.PrevHash)

			if gotCanonicalHex != c.ExpectedCanonicalBytesHex {
				t.Errorf(
					"canonical bytes drift\n  case: %s\n  recorded: %s\n  computed: %s",
					c.Description, c.ExpectedCanonicalBytesHex, gotCanonicalHex,
				)
			}
			if gotHashHex != c.ExpectedHashHex {
				t.Errorf(
					"hash drift\n  case: %s\n  recorded: %s\n  computed: %s",
					c.Description, c.ExpectedHashHex, gotHashHex,
				)
			}

			// 3. Assert hash.Compute on the same event yields the same hash —
			//    this protects against drift in the higher-level wrapper as
			//    well as in canonicalize itself.
			var ev hash.Event
			if err := json.Unmarshal(c.Event, &ev); err != nil {
				t.Fatalf("unmarshal event for hash.Compute cross-check: %v", err)
			}
			ev.PrevHash = c.PrevHash
			h, err := hash.Compute(ev)
			if err != nil {
				t.Fatalf("hash.Compute: %v", err)
			}
			if h != c.ExpectedHashHex {
				t.Errorf("hash.Compute drift\n  case: %s\n  recorded: %s\n  computed: %s", c.Description, c.ExpectedHashHex, h)
			}
		})
	}
}

func regenerateGolden(t *testing.T) {
	t.Helper()

	cases := seedCases()
	for i := range cases {
		c := &cases[i]

		// Resolve chained PrevHash before computing this case.
		if c.ChainPrevFromIndex != nil {
			c.PrevHash = cases[*c.ChainPrevFromIndex].ExpectedHashHex
			c.Event = renderEventForChainedCase(c.PrevHash)
		}

		// Inject the resolved PrevHash into the event body so the recorded
		// "event" matches the recorded "prev_hash" exactly. (For chained
		// cases this was already done above; for the rest it's a no-op
		// since the seed already encoded the same prev_hash.)
		var eventMap map[string]any
		if err := json.Unmarshal(c.Event, &eventMap); err != nil {
			t.Fatalf("seed case %d unmarshal: %v", i, err)
		}
		eventMap["prev_hash"] = c.PrevHash
		reEncoded, err := json.Marshal(eventMap)
		if err != nil {
			t.Fatalf("seed case %d remarshal: %v", i, err)
		}
		c.Event = reEncoded

		canonicalHex, hashHex := computeExpected(t, c.Event, c.PrevHash)
		c.ExpectedCanonicalBytesHex = canonicalHex
		c.ExpectedHashHex = hashHex
	}

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshal golden: %v", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(goldenPath, out, 0o644); err != nil {
		t.Fatalf("write golden: %v", err)
	}
	t.Logf("regenerated %d cases at %s", len(cases), goldenPath)
}
