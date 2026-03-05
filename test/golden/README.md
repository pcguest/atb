# ATB Golden Test - Cross-Language Consistency

## Purpose

Ensure Go, Python, and TypeScript produce byte-identical RFC 8785 canonical JSON
and SHA-256 hashes for the same event payload.

This protects ATB's core portability guarantee: traces generated in one language
must verify in every other language.

## Files

- `input.json`: language-agnostic fixture
- `golden_test.go`: Go baseline generator + canonical/hash assertions
- `encrypt_parity_test.go`: Go/Python/TypeScript AES-GCM parity assertions
- `encrypt_parity.py`: Python parity helper (invoked by Go test)
- `encrypt_parity.js`: TypeScript parity helper (invoked by Go test)
- `encrypt-vector.hex`: deterministic encryption golden fixture
- `verify.py`: Python canonical/hash parity check against Go outputs
- `verify.js`: TypeScript canonical/hash parity check against Go outputs

## Run Locally

```bash
cd test/golden

# 1) Generate Go baseline outputs
GOCACHE=/tmp/atb-go-cache go test -v -run TestGoldenCanonicalization

# 2) Verify Python parity
python3 verify.py

# 3) Verify TypeScript parity (after sdk/typescript build)
node verify.js

# 4) Optional manual diffs
diff output-go.json output-python.json
diff output-python.json output-typescript.json
diff hash-go.txt hash-python.txt
diff hash-python.txt hash-typescript.txt

# 5) Cross-language encryption parity
GOCACHE=/tmp/atb-go-cache go test -v -run TestEncryptParity_AllSDKs
```

## Expected Canonical Output

```json
{"actor":"golden-test","data":{"model":"gpt-4","nested":{"array":[1,2,3],"bool":true,"null":null,"unicode":"Hello 世界 🌍"},"prompt":"What is 2+2?","temperature":0},"ts":"2026-03-03T00:00:00Z","type":"agent.think"}
```

## Updating the Fixture

If `input.json` changes:

1. Run Go test and capture new canonical/hash values.
2. Update `expectedCanonical` and `expectedHash` in `golden_test.go`.
3. Re-run Python and TypeScript verifiers.
4. Ensure CI passes before merge.

## CI Integration

`golden-test` runs in `.github/workflows/ci.yml` on every PR and push to `main`.
Any mismatch fails CI.
