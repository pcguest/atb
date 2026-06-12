// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"strings"
	"testing"
)

// The capture proxy is an HTTPS forward proxy; it has no /openai or
// /anthropic path routing, so the startup hints and help text must never
// suggest provider base-URL overrides.
func TestInterceptEnvHintsDescribeForwardProxy(t *testing.T) {
	var buf bytes.Buffer
	printInterceptEnvHints(&buf, 8080)
	out := buf.String()

	for _, want := range []string{
		"HTTPS_PROXY=http://127.0.0.1:8080",
		"SSL_CERT_FILE=",
		"NODE_EXTRA_CA_CERTS=",
		"base-URL path overrides are not supported",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("env hints missing %q in output:\n%s", want, out)
		}
	}

	for _, forbidden := range []string{"BASE_URL", "localhost:8080/openai", "localhost:8080/anthropic", "/openai", "/anthropic"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("env hints still contain misleading %q in output:\n%s", forbidden, out)
		}
	}
}

func TestInterceptHelpHasNoBaseURLClaim(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runInterceptCommand([]string{"--help"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("--help exit = %d, want %d (stderr: %s)", code, exitSuccess, stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, "BASE_URL") {
		t.Errorf("help output still mentions BASE_URL:\n%s", out)
	}
	if !strings.Contains(out, "HTTPS_PROXY") {
		t.Errorf("help output does not mention HTTPS_PROXY routing:\n%s", out)
	}
}
