//go:build integration

package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

const repoRoot = "/Users/paddyguest/atb"

func TestPythonSDKBundleCompatibility(t *testing.T) {
	tempDir := t.TempDir()

	cmd := exec.Command("python3", "-c", `
import sys, os
sys.path.insert(0, 'sdk/python')
from atb import Bundle
b = Bundle()
b.append('dev.session', {'sdk': 'python', 'test': True})
b.save(os.path.join(sys.argv[1], 'bundle.atb'))
`, tempDir)
	cmd.Dir = repoRoot

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("python bundle write failed: %s", stderr.String())
	}

	b, err := bundle.Load(filepath.Join(tempDir, "bundle.atb"))
	if err != nil {
		t.Fatalf("load python bundle: %v", err)
	}

	assertBundleHasSessionEvent(t, b)
}

func TestTypeScriptSDKBundleCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	tsEntry := filepath.Join(repoRoot, "sdk", "typescript", "dist", "index.js")
	if _, err := os.Stat(tsEntry); err != nil {
		// The TypeScript package ships a CLI stub only. Without compiled output,
		// this test would need a checked-in build artifact or a real bundle-write CLI path.
		t.Skip("TypeScript SDK does not expose a CLI bundle-write path; test manually per sdk/typescript/PUBLISHING.md")
	}

	cmd := exec.Command("node", "-e", `
const { Bundle } = require("./sdk/typescript/dist/index.js");
const path = require("node:path");
const b = new Bundle();
b.append("dev.session", { sdk: "typescript", test: true });
b.save(path.join(process.argv[1], "bundle.atb"));
`, tempDir)
	cmd.Dir = repoRoot

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("typescript bundle write failed: %s", stderr.String())
	}

	b, err := bundle.Load(filepath.Join(tempDir, "bundle.atb"))
	if err != nil {
		t.Fatalf("load typescript bundle: %v", err)
	}

	assertBundleHasSessionEvent(t, b)
}

func assertBundleHasSessionEvent(t *testing.T, b *bundle.Bundle) {
	t.Helper()

	if err := b.Verify(); err != nil {
		t.Fatalf("verify bundle: %v", err)
	}
	if len(b.Records) < 2 {
		t.Fatalf("expected at least 2 records, got %d", len(b.Records))
	}

	for _, record := range b.Records {
		if record.Event.Type == "dev.session" {
			return
		}
	}

	t.Fatalf("expected at least one dev.session record")
}
