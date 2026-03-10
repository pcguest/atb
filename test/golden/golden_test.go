package golden

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/pcguest/atb/internal/canonicalize"
)

const (
	genesisHash       = "0000000000000000000000000000000000000000000000000000000000000000"
	expectedCanonical = `{"actor":"golden-test","data":{"model":"gpt-4","nested":{"array":[1,2,3],"bool":true,"null":null,"unicode":"Hello 世界 🌍"},"prompt":"What is 2+2?","temperature":0},"ts":"2026-03-03T00:00:00Z","type":"agent.think"}`
	expectedHash      = "8df8de142c5227c5f8024dcc79b1057654dd78ac683650d5051cb2f960d1a7a8"
)

func TestGoldenCanonicalization(t *testing.T) {
	input, err := os.ReadFile("input.json")
	if err != nil {
		t.Fatalf("read input.json: %v", err)
	}

	canonical, err := canonicalize.MarshalRaw(input)
	if err != nil {
		t.Fatalf("canonicalize input: %v", err)
	}

	h := sha256.New()
	h.Write([]byte(genesisHash))
	h.Write(canonical)
	sum := hex.EncodeToString(h.Sum(nil))

	if err := os.WriteFile("output-go.json", canonical, 0o644); err != nil {
		t.Fatalf("write output-go.json: %v", err)
	}
	if err := os.WriteFile("hash-go.txt", []byte(sum), 0o644); err != nil {
		t.Fatalf("write hash-go.txt: %v", err)
	}

	t.Logf("go canonical: %s", string(canonical))
	t.Logf("go hash: %s", sum)

	var parsed map[string]any
	if err := json.Unmarshal(canonical, &parsed); err != nil {
		t.Fatalf("canonical output is invalid json: %v", err)
	}

	if string(canonical) != expectedCanonical {
		t.Fatalf("canonical mismatch\ngot:  %s\nwant: %s", string(canonical), expectedCanonical)
	}

	if expectedHash != "" && sum != expectedHash {
		t.Fatalf("hash mismatch\ngot:  %s\nwant: %s", sum, expectedHash)
	}
}
