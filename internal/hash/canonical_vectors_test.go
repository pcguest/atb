// SPDX-License-Identifier: MIT
package hash_test

// TestGoldenVectors pins the byte-for-byte canonical encoding and SHA-256
// hash for a fixed set of Event objects shared across the Go runtime and
// the Python and TypeScript SDKs.
//
// This is the cross-language contract test. The same testdata file is
// loaded by:
//   - sdk/python/tests/test_canonical_hash.py
//   - sdk/typescript/src/canonical_hash.test.ts
//
// Any change that causes this test to fail is a BREAKING SCHEMA CHANGE.
// To regenerate after a deliberate, vetted format change:
//
//	go test ./internal/hash/... -run TestGoldenVectors -update-vectors

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/pcguest/atb/internal/canonicalize"
)

var updateVectors = flag.Bool(
	"update-vectors",
	false,
	"regenerate internal/hash/testdata/canonical_vectors.json from the live Go canonicaliser",
)

const canonicalVectorsPath = "testdata/canonical_vectors.json"

type canonicalVector struct {
	Description            string          `json:"description"`
	Event                  json.RawMessage `json:"event"`
	PrevHash               string          `json:"prev_hash"`
	ExpectedCanonicalBytes string          `json:"expected_canonical_bytes"`
	ExpectedHash           string          `json:"expected_hash"`
}

const vectorsGenesis = "0000000000000000000000000000000000000000000000000000000000000000"

// seedVectors returns the input cases. Expected outputs are computed by the
// Go canonicaliser (the source of truth) and serialised to disk.
func seedVectors() []canonicalVector {
	return []canonicalVector{
		{
			Description: "minimal event: type, seq, timestamp, hash_algo, prev_hash, data=null",
			Event: mustCompact(`{
				"seq": 1,
				"prev_hash": "` + vectorsGenesis + `",
				"type": "test.minimal",
				"hash_algo": "sha256",
				"timestamp": "2026-01-02T03:04:05Z",
				"data": null
			}`),
			PrevHash: vectorsGenesis,
		},
		{
			Description: "all top-level fields set with a small data object",
			Event: mustCompact(`{
				"seq": 7,
				"prev_hash": "1111111111111111111111111111111111111111111111111111111111111111",
				"type": "test.all_fields",
				"hash_algo": "sha256",
				"timestamp": "2026-01-02T03:04:05Z",
				"trace_id": "0123456789abcdef0123456789abcdef",
				"span_id": "0123456789abcdef",
				"parent_span_id": "fedcba9876543210",
				"actor_id": "alice",
				"org_id": "acme",
				"workspace_id": "ws-1",
				"data": {"key": "value", "n": 42}
			}`),
			PrevHash: "1111111111111111111111111111111111111111111111111111111111111111",
		},
		{
			Description: "data is null",
			Event: mustCompact(`{
				"seq": 2,
				"prev_hash": "` + vectorsGenesis + `",
				"type": "test.data_null",
				"hash_algo": "sha256",
				"timestamp": "2026-01-02T03:04:05Z",
				"data": null
			}`),
			PrevHash: vectorsGenesis,
		},
		{
			Description: "data is the float 1.0 (number-as-data float serialisation)",
			Event: mustCompact(`{
				"seq": 3,
				"prev_hash": "` + vectorsGenesis + `",
				"type": "test.data_float_one",
				"hash_algo": "sha256",
				"timestamp": "2026-01-02T03:04:05Z",
				"data": 1.0
			}`),
			PrevHash: vectorsGenesis,
		},
		{
			Description: "data is the integer 2^53 (boundary of JS-safe integers)",
			Event: mustCompact(`{
				"seq": 4,
				"prev_hash": "` + vectorsGenesis + `",
				"type": "test.data_pow2_53",
				"hash_algo": "sha256",
				"timestamp": "2026-01-02T03:04:05Z",
				"data": 9007199254740992
			}`),
			PrevHash: vectorsGenesis,
		},
		{
			Description: "atb.bundle.manifest with data as JSON-encoded string (double-encoded, manifest v1 wire form)",
			Event: mustCompact(`{
				"seq": 0,
				"prev_hash": "` + vectorsGenesis + `",
				"type": "atb.bundle.manifest",
				"hash_algo": "sha256",
				"timestamp": "2026-01-02T03:04:05Z",
				"data": "{\"version\":1,\"created_at\":\"2026-01-02T03:04:05Z\",\"bundle_id\":\"01HV0000000000000000000000\"}"
			}`),
			PrevHash: vectorsGenesis,
		},
		{
			Description: "atb.bundle.manifest v2 with structured-object data (no double-encoding)",
			Event: mustCompact(`{
				"seq": 0,
				"prev_hash": "` + vectorsGenesis + `",
				"type": "atb.bundle.manifest",
				"hash_algo": "sha256",
				"timestamp": "2026-01-02T03:04:05Z",
				"data": {
					"version": 2,
					"created_at": "2026-01-02T03:04:05Z",
					"bundle_id": "00112233445566778899aabbccddeeff"
				}
			}`),
			PrevHash: vectorsGenesis,
		},
	}
}

func mustCompact(s string) json.RawMessage {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		panic(fmt.Sprintf("seedVectors: invalid JSON literal: %v\n%s", err, s))
	}
	return buf.Bytes()
}

func computeVector(t *testing.T, eventJSON json.RawMessage, prevHash string) (canonical []byte, hashHex string) {
	t.Helper()
	canonical, err := canonicalize.MarshalRaw(eventJSON)
	if err != nil {
		t.Fatalf("canonicalise: %v", err)
	}
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write(canonical)
	return canonical, hex.EncodeToString(h.Sum(nil))
}

func TestGoldenVectors(t *testing.T) {
	if *updateVectors {
		regenerateCanonicalVectors(t)
		return
	}

	raw, err := os.ReadFile(canonicalVectorsPath)
	if err != nil {
		t.Fatalf(
			"golden vectors missing or unreadable at %s: %v\n"+
				"To bootstrap or regenerate after a vetted format change, run:\n"+
				"  go test ./internal/hash/... -run TestGoldenVectors -update-vectors",
			canonicalVectorsPath, err,
		)
	}

	var vectors []canonicalVector
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("parse golden vectors: %v", err)
	}

	seed := seedVectors()
	if len(vectors) != len(seed) {
		t.Fatalf("vector count (%d) != seed count (%d); regenerate with -update-vectors",
			len(vectors), len(seed))
	}

	for i, v := range vectors {
		t.Run(fmt.Sprintf("vector_%02d", i), func(t *testing.T) {
			expectedCanonical, err := base64.StdEncoding.DecodeString(v.ExpectedCanonicalBytes)
			if err != nil {
				t.Fatalf("decode expected_canonical_bytes: %v", err)
			}

			gotCanonical, gotHash := computeVector(t, v.Event, v.PrevHash)

			if !bytes.Equal(gotCanonical, expectedCanonical) {
				diffIdx := firstDiff(gotCanonical, expectedCanonical)
				t.Errorf(
					"canonical bytes drift\n  case: %s\n  first diff at byte %d\n  expected: %s\n  got:      %s",
					v.Description, diffIdx,
					hex.EncodeToString(expectedCanonical),
					hex.EncodeToString(gotCanonical),
				)
			}
			if gotHash != v.ExpectedHash {
				t.Errorf(
					"hash drift\n  case: %s\n  expected: %s\n  got:      %s",
					v.Description, v.ExpectedHash, gotHash,
				)
			}
		})
	}
}

func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func regenerateCanonicalVectors(t *testing.T) {
	t.Helper()

	vectors := seedVectors()
	for i := range vectors {
		v := &vectors[i]
		canonical, hashHex := computeVector(t, v.Event, v.PrevHash)
		v.ExpectedCanonicalBytes = base64.StdEncoding.EncodeToString(canonical)
		v.ExpectedHash = hashHex
		t.Logf("vector %d hash = %s", i, hashHex)
	}

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	out, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		t.Fatalf("marshal vectors: %v", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(canonicalVectorsPath, out, 0o644); err != nil {
		t.Fatalf("write vectors: %v", err)
	}

	// Sanity: confirm hashes are not all identical.
	seen := make(map[string]struct{})
	for _, v := range vectors {
		seen[v.ExpectedHash] = struct{}{}
	}
	if len(seen) != len(vectors) {
		t.Fatalf("regenerated vectors contain duplicate hashes (%d unique of %d)", len(seen), len(vectors))
	}
	t.Logf("wrote %d unique vectors to %s", len(vectors), canonicalVectorsPath)
}
