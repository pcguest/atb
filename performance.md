# Performance Baseline

## 1. Overview

These figures are the current baseline for ATB bundle operations on a single-core Apple M2 system. All measurements are local I/O only, with no network calls and no daemon in the loop. Numbers will differ on server hardware, so treat them as relative reference points rather than absolute guarantees.

## 2. Append throughput

| Operation | Events | Time/op | Throughput | Allocs/op |
|-----------|--------|---------|------------|-----------|
| `BundleAppend100` | 100 | 2.48ms | ~40k ops/s | 8837 |
| `BundleAppend1000` | 1000 | 23.3ms | ~43k ops/s | 89354 |

Throughput is stable between 100 and 1000 events. Append cost is linear, with no observable per-bundle overhead.

## 3. Read throughput

| Operation | Events | Time/op | Allocs/op |
|-----------|--------|---------|-----------|
| `BundleRead100` | 100 | 638µs | 2328 |
| `BundleRead1000` | 1000 | 7.2ms | 23032 |

Read performance scales linearly across the measured bundle sizes.

## 4. Chain verification

| Operation | Events | Time/op | Allocs/op |
|-----------|--------|---------|-----------|
| `VerifyChain100` | 100 | 2.4ms | 13237 |
| `VerifyChain1000` | 1000 | 25ms | 131849 |

Verification is more allocation-intensive than read alone because it computes SHA-256 and RFC 8785 canonicalisation per record.

## 5. Benchmark environment

Hardware: Apple M2, `darwin/arm64`.

Go benchmark flags: `-benchtime=3s`.

These figures will be updated when a server-class baseline is available.

## 6. How to reproduce

```bash
go test -bench=. -benchtime=3s ./internal/bundle/...
```

`internal/bundle/bench_test.go` must be present. It is currently untracked, so run `git checkout` or recreate it if needed before benchmarking.
