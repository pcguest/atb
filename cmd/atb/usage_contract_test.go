// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCommandUsageRenderers(t *testing.T) {
	var writerOutput bytes.Buffer
	printEventsUsage(&writerOutput)
	printInspectCommandUsage(&writerOutput)
	printKeygenCommandUsage(&writerOutput)
	printProfilesUsage(&writerOutput)
	printProfilesValidateUsage(&writerOutput)
	printSignCommandUsage(&writerOutput)
	for _, want := range []string{
		"atb events", "atb inspect", "atb keygen", "atb profiles",
		"atb profiles validate", "atb sign",
	} {
		if !strings.Contains(writerOutput.String(), want) {
			t.Fatalf("writer usage output missing %q: %q", want, writerOutput.String())
		}
	}

	processOutput := captureProcessStdout(t, func() {
		printArchiveUsage()
		printDocUsage()
		printUsageJSON()
		printUsage()
		printTrustReportUsage()
		printViewUsage()
	})
	for _, want := range []string{
		"atb archive", "atb doc", `"commands"`, "ATB — Agent Trace Bundle",
		"atb trust-report", "atb view",
	} {
		if !strings.Contains(processOutput, want) {
			t.Fatalf("process usage output missing %q", want)
		}
	}
}

func TestShortEvidenceHashContracts(t *testing.T) {
	if shortEvidenceHash("") != "-" {
		t.Fatal("empty hash not replaced")
	}
	if shortEvidenceHash("short") != "short" {
		t.Fatal("short hash changed")
	}
	if got := shortEvidenceHash(strings.Repeat("a", 20)); got != strings.Repeat("a", 16)+"…" {
		t.Fatalf("long hash=%q", got)
	}
}

func captureProcessStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	output, err := os.CreateTemp(t.TempDir(), "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = output
	defer func() {
		os.Stdout = original
		_ = output.Close()
	}()

	fn()
	if err := output.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
