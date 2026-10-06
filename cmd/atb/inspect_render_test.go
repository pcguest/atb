// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestSanitizeForTerminal(t *testing.T) {
	cases := map[string]string{
		"\x1b[31mred\x1b[0m": `\x1b[31mred\x1b[0m`,
		"a\tb":               "a b",
		"plain text":         "plain text",
		"\x00\x07":           `\x00\x07`,
		"\u0085next":         `\x85next`,
		"\u202eRTL":          `\u202eRTL`,
		"\u2028line":         `\u2028line`,
	}
	for in, want := range cases {
		if got := sanitizeForTerminal(in); got != want {
			t.Errorf("sanitizeForTerminal(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestInspectTableSanitizesControlCharacters proves attacker-controlled
// evidence cannot inject terminal escape sequences through the inspect table.
func TestInspectTableSanitizesControlCharacters(t *testing.T) {
	b, err := bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV3})
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	if err := b.Append("test.escape", "\x1b[31mPWNED\x1b[0m"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Append("test.html", map[string]any{"html": "<script>alert(1)</script>"}); err != nil {
		t.Fatalf("append: %v", err)
	}

	var buf bytes.Buffer
	if err := renderInspectTable(&buf, b); err != nil {
		t.Fatalf("renderInspectTable: %v", err)
	}
	out := buf.String()
	if strings.ContainsRune(out, 0x1b) {
		t.Fatalf("ESC control character leaked into inspect output: %q", out)
	}
	if !strings.Contains(out, `\x1b[31m`) {
		t.Fatalf("expected the escape to be rendered visibly, got %q", out)
	}
}

// TestInspectJSONEscapesBidiControls proves --json output cannot spoof a
// terminal: bidi overrides are emitted as \uXXXX escapes while remaining valid
// JSON.
func TestInspectJSONEscapesBidiControls(t *testing.T) {
	b, err := bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV3})
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	if err := b.Append("test.bidi", map[string]any{"note": "safe\u202eevil"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	path := filepath.Join(t.TempDir(), "bidi.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	var buf bytes.Buffer
	if code := runInspect([]string{"--bundle", path, "--json"}, &buf, io.Discard); code != exitSuccess {
		t.Fatalf("runInspect exit = %d", code)
	}
	out := buf.String()
	if strings.ContainsRune(out, 0x202e) {
		t.Fatalf("raw bidi override leaked into JSON output")
	}
	if !strings.Contains(out, `\u202e`) {
		t.Fatalf("expected escaped bidi override, got %q", out)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("escaped output is not valid JSON: %q", out)
	}
}
