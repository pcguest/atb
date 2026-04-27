// SPDX-License-Identifier: MIT
package bundle_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
)

var (
	benchBundleSink *bundle.Bundle
	benchReportSink verify.Report
)

func BenchmarkBundleAppend100(b *testing.B) {
	benchmarkBundleAppend(b, 100)
}

func BenchmarkBundleAppend1000(b *testing.B) {
	benchmarkBundleAppend(b, 1000)
}

func BenchmarkBundleRead100(b *testing.B) {
	benchmarkBundleRead(b, 100)
}

func BenchmarkBundleRead1000(b *testing.B) {
	benchmarkBundleRead(b, 1000)
}

func BenchmarkVerifyChain100(b *testing.B) {
	benchmarkVerifyChain(b, 100)
}

func BenchmarkVerifyChain1000(b *testing.B) {
	benchmarkVerifyChain(b, 1000)
}

func benchmarkBundleAppend(b *testing.B, eventCount int) {
	b.Helper()

	dir := b.TempDir()
	path := filepath.Join(dir, "bundle.atb")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bundleFile, err := bundle.New()
		if err != nil {
			b.Fatalf("create bundle: %v", err)
		}
		appendBenchmarkRequestEvents(b, bundleFile, eventCount)
		if err := bundleFile.Save(path); err != nil {
			b.Fatalf("save bundle: %v", err)
		}
		benchBundleSink = bundleFile
	}

	reportAppendMetrics(b, eventCount)
}

func benchmarkBundleRead(b *testing.B, eventCount int) {
	b.Helper()

	dir := b.TempDir()
	path := writeBenchmarkBundle(b, dir, eventCount)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		loaded, err := bundle.Load(path)
		if err != nil {
			b.Fatalf("load bundle: %v", err)
		}
		benchBundleSink = loaded
	}
}

func benchmarkVerifyChain(b *testing.B, eventCount int) {
	b.Helper()

	dir := b.TempDir()
	path := writeBenchmarkBundle(b, dir, eventCount)
	loaded, err := bundle.Load(path)
	if err != nil {
		b.Fatalf("load bundle: %v", err)
	}

	initial := verify.Verify(loaded, path, "")
	if !initial.Integrity.ChainValid {
		b.Fatalf("initial verify: chain invalid: %+v", initial.Integrity)
	}
	if len(initial.Profiles) != 0 {
		b.Fatalf("initial verify: expected no matched profiles, got %d", len(initial.Profiles))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		report := verify.Verify(loaded, path, "")
		if !report.Integrity.ChainValid {
			b.Fatalf("verify chain: chain invalid: %+v", report.Integrity)
		}
		benchReportSink = report
	}
}

func writeBenchmarkBundle(b testing.TB, dir string, eventCount int) string {
	b.Helper()

	path := filepath.Join(dir, fmt.Sprintf("bundle-%d.atb", eventCount))
	bundleFile, err := bundle.New()
	if err != nil {
		b.Fatalf("create bundle: %v", err)
	}
	appendBenchmarkRequestEvents(b, bundleFile, eventCount)
	if err := bundleFile.Save(path); err != nil {
		b.Fatalf("save bundle: %v", err)
	}
	return path
}

func appendBenchmarkRequestEvents(b testing.TB, bundleFile *bundle.Bundle, eventCount int) {
	b.Helper()

	for i := 0; i < eventCount; i++ {
		if err := bundleFile.Append("ai.request.received", benchmarkRequestPayload(i)); err != nil {
			b.Fatalf("append ai.request.received: %v", err)
		}
	}
}

func benchmarkRequestPayload(i int) map[string]any {
	return map[string]any{
		"request_id":    fmt.Sprintf("req-%06d", i),
		"actor_id_hash": fmt.Sprintf("actor-%08d", i%1024),
		"purpose_tag":   "benchmark_unmatched",
	}
}

func reportAppendMetrics(b *testing.B, eventCount int) {
	totalAppends := float64(eventCount * b.N)
	if totalAppends == 0 {
		return
	}
	elapsed := b.Elapsed()
	if elapsed <= 0 {
		return
	}
	b.ReportMetric(float64(elapsed.Nanoseconds())/totalAppends, "ns/append")
	b.ReportMetric(totalAppends/elapsed.Seconds(), "appends/s")
}
