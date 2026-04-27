// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	evidencepkg "github.com/pcguest/atb/internal/evidence"
)

func TestRunEvidenceMissingBundleFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidence(nil, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("runEvidence() exit code = %d, want %d", exitCode, exitUserError)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--bundle is required") {
		t.Fatalf("stderr = %q, want missing bundle error", stderr.String())
	}
}

func TestRunEvidenceHealthyText(t *testing.T) {
	bundlePath := signedCLIEvidenceBundle(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidence([]string{"--bundle", bundlePath}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvidence() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"Bundle:",
		"Tampered:  false",
		"Manifest:  version=1",
		"Records:",
		"Signatures:",
		"backend=local",
		"valid=true",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("text output missing %q:\n%s", want, output)
		}
	}
}

func TestRunEvidenceHealthyJSONMatchesBuild(t *testing.T) {
	bundlePath := signedCLIEvidenceBundle(t)
	want, err := evidencepkg.Build(t.Context(), bundlePath)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidence([]string{"--bundle", bundlePath, "--format", "json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runEvidence() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	if strings.HasSuffix(stdout.String(), "\n") {
		t.Fatalf("JSON output has trailing newline: %q", stdout.String())
	}

	var got evidencepkg.BundleEvidence
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal JSON evidence: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON evidence mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestRunEvidenceTamperedJSON(t *testing.T) {
	bundlePath := signedCLIEvidenceBundle(t)
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	tampered := bytes.Replace(raw, []byte("approve-change"), []byte("changed-value"), 1)
	if bytes.Equal(tampered, raw) {
		t.Fatal("failed to tamper bundle")
	}
	if err := os.WriteFile(bundlePath, tampered, 0o600); err != nil {
		t.Fatalf("write tampered bundle: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runEvidence([]string{"--bundle", bundlePath, "--format", "json"}, &stdout, &stderr)
	if exitCode != exitIntegrityFailure {
		t.Fatalf("runEvidence() exit code = %d, want %d (stderr=%q)", exitCode, exitIntegrityFailure, stderr.String())
	}

	var got evidencepkg.BundleEvidence
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal tampered evidence: %v", err)
	}
	if !got.Tampered {
		t.Fatal("tampered evidence has tampered=false")
	}
	if len(got.Signatures) == 0 {
		t.Fatal("tampered evidence missing signatures")
	}
}

func signedCLIEvidenceBundle(t testing.TB) string {
	t.Helper()

	bundlePath := writeVerifyTestBundle(t, buildSignSCBundle(t))
	keyPath := writeSignTestPrivateKey(t, t.TempDir())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSign([]string{"--bundle", bundlePath, "--key", keyPath}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runSign() exit code = %d, want %d (stderr=%q)", exitCode, exitSuccess, stderr.String())
	}
	return bundlePath
}
