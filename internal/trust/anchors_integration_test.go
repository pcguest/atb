//go:build integration

package trust

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestVerifyAnchors_RealAnchorRoundTrip(t *testing.T) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("skip integration test: go toolchain unavailable: %v", err)
	}

	bundlePath := filepath.Join(t.TempDir(), "run.atb", bundle.BundleFile)
	b := newTrustTestBundle(t)
	if err := b.Append(event.TypeDevSession, map[string]any{"event_id": "anchor-round-trip"}); err != nil {
		t.Fatalf("append dev session: %v", err)
	}
	if err := b.Save(bundlePath); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		goBinary,
		"run",
		"./cmd/atb",
		"anchor",
		bundlePath,
		"--tsa-url",
		"https://freetsa.org/tsr",
	)
	cmd.Dir = integrationRepoRoot(t)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Skip("skip integration test: public TSA request timed out")
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if shouldSkipAnchorIntegration(message) {
			t.Skipf("skip integration test: public TSA unreachable: %s", message)
		}
		t.Fatalf("run anchor command: %v\n%s", err, message)
	}

	loaded, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("load anchored bundle: %v", err)
	}

	results := VerifyAnchors(loaded)
	if len(results) == 0 {
		t.Fatal("expected at least one anchor verification result")
	}
	for _, result := range results {
		if result.TSAVerified {
			return
		}
	}

	t.Fatalf("expected at least one verified anchor result, got %+v", results)
}

func integrationRepoRoot(t testing.TB) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repo root: runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func shouldSkipAnchorIntegration(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "anchor: http post:") ||
		strings.Contains(lower, "anchor: tsa returned") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "temporary failure")
}
