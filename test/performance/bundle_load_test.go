package performance

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

func BenchmarkBundleLoad100Blocks(b *testing.B) {
	benchLoadBundle(b, 100)
}

func BenchmarkBundleLoad1000Blocks(b *testing.B) {
	benchLoadBundle(b, 1000)
}

func BenchmarkBundleLoad10000Blocks(b *testing.B) {
	benchLoadBundle(b, 10000)
}

func benchLoadBundle(b *testing.B, numBlocks int) {
	bundlePath := buildBenchmarkBundle(b, numBlocks)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		loaded, err := bundle.Load(bundlePath)
		if err != nil {
			b.Fatalf("load benchmark bundle (%d blocks): %v", numBlocks, err)
		}
		if err := loaded.Verify(); err != nil {
			b.Fatalf("verify benchmark bundle (%d blocks): %v", numBlocks, err)
		}

		elapsed := time.Since(start)
		if elapsed > 2*time.Second {
			b.Errorf("bundle load > 2s for %d blocks: %v", numBlocks, elapsed)
		}
	}
}

func buildBenchmarkBundle(b *testing.B, numBlocks int) string {
	b.Helper()

	tmpDir := b.TempDir()
	bundlePath := filepath.Join(tmpDir, fmt.Sprintf("benchmark-%d.atb", numBlocks))
	bun := bundle.New()

	for i := 0; i < numBlocks; i++ {
		if err := bun.Append("benchmark.load", map[string]interface{}{
			"index":     i + 1,
			"timestamp": time.Unix(int64(i), 0).UTC().Format(time.RFC3339Nano),
			"payload":   fmt.Sprintf("block-%d", i+1),
		}); err != nil {
			b.Fatalf("append benchmark block %d: %v", i+1, err)
		}
	}

	if err := bun.Save(bundlePath); err != nil {
		b.Fatalf("save benchmark bundle: %v", err)
	}

	return bundlePath
}
