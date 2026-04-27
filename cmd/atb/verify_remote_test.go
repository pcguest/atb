// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/push"
)

// bundleNDJSON serialises a bundle to NDJSON bytes suitable for faking S3 GetObject.
func bundleNDJSON(t *testing.T, b *bundle.Bundle) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, r := range b.Records {
		if err := enc.Encode(r); err != nil {
			t.Fatalf("encode bundle record: %v", err)
		}
	}
	return buf.Bytes()
}

// setRemoteUploader installs fake as the process-global remote verify uploader
// and restores the original on t.Cleanup.
func setRemoteUploader(t *testing.T, fake push.S3Uploader) {
	t.Helper()
	prev := remoteVerifyUploader
	remoteVerifyUploader = fake
	t.Cleanup(func() { remoteVerifyUploader = prev })
}

// ── happy path ────────────────────────────────────────────────────────────────

func TestRunVerifyRemoteHappyPath(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "atb.snapshot", map[string]string{"name": "test"})
	headHash := b.Records[len(b.Records)-1].Hash
	body := bundleNDJSON(t, b)

	remoteKey := "prefix/sha256-" + headHash + ".atb"
	remoteURI := "s3://my-bucket/" + remoteKey
	fake := &fakeS3Uploader{getBody: body}
	setRemoteUploader(t, fake)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--remote", remoteURI}, &stdout, &stderr)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr=%q stdout=%q)", exitSuccess, code, stderr.String(), stdout.String())
	}
	if len(fake.getCalls) != 1 {
		t.Fatalf("expected 1 GetObject call, got %d", len(fake.getCalls))
	}
	if fake.getCalls[0].Bucket != "my-bucket" {
		t.Fatalf("bucket: got %q want my-bucket", fake.getCalls[0].Bucket)
	}
	if fake.getCalls[0].Key != remoteKey {
		t.Fatalf("key: got %q want %q", fake.getCalls[0].Key, remoteKey)
	}
	// Text output must contain remote URI and key hash OK line.
	out := stdout.String()
	if !strings.Contains(out, "s3://my-bucket/") {
		t.Fatalf("stdout missing remote URI, got: %q", out)
	}
	if !strings.Contains(out, "OK") {
		t.Fatalf("stdout missing 'OK' key hash result, got: %q", out)
	}
}

func TestRunVerifyRemoteJSON(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "atb.snapshot", map[string]string{"name": "test"})
	headHash := b.Records[len(b.Records)-1].Hash
	body := bundleNDJSON(t, b)

	remoteURI := "s3://my-bucket/prefix/sha256-" + headHash + ".atb"
	fake := &fakeS3Uploader{getBody: body}
	setRemoteUploader(t, fake)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--remote", remoteURI, "--format", "json"}, &stdout, &stderr)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr=%q)", exitSuccess, code, stderr.String())
	}

	var rep remoteVerifyReport
	if err := json.NewDecoder(&stdout).Decode(&rep); err != nil {
		t.Fatalf("decode remote report: %v", err)
	}
	if rep.RemoteURI != remoteURI {
		t.Fatalf("remote_uri: got %q want %q", rep.RemoteURI, remoteURI)
	}
	if !rep.KeyHashOK {
		t.Fatalf("key_hash_ok should be true; warn=%q", rep.KeyHashWarn)
	}
}

// ── key hash mismatch ─────────────────────────────────────────────────────────

func TestRunVerifyRemoteKeyHashMismatch(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "atb.snapshot", map[string]string{"name": "real"})
	body := bundleNDJSON(t, b)

	// Use a wrong hash in the key name.
	wrongHash := strings.Repeat("0", 64)
	remoteURI := "s3://my-bucket/prefix/sha256-" + wrongHash + ".atb"
	fake := &fakeS3Uploader{getBody: body}
	setRemoteUploader(t, fake)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--remote", remoteURI, "--format", "json"}, &stdout, &stderr)

	// Must be integrity failure.
	if code != exitIntegrityFailure {
		t.Fatalf("expected exit code %d (integrity failure), got %d", exitIntegrityFailure, code)
	}

	var rep remoteVerifyReport
	if err := json.NewDecoder(&stdout).Decode(&rep); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rep.KeyHashOK {
		t.Fatalf("key_hash_ok should be false on hash mismatch")
	}
	if !strings.Contains(rep.KeyHashWarn, "CRITICAL") {
		t.Fatalf("key_hash_warn should mention CRITICAL, got %q", rep.KeyHashWarn)
	}
}

func TestRunVerifyRemoteKeyHashMismatchText(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "atb.snapshot", map[string]string{"name": "real"})
	body := bundleNDJSON(t, b)

	wrongHash := strings.Repeat("0", 64)
	remoteURI := "s3://my-bucket/prefix/sha256-" + wrongHash + ".atb"
	fake := &fakeS3Uploader{getBody: body}
	setRemoteUploader(t, fake)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--remote", remoteURI}, &stdout, &stderr)

	if code != exitIntegrityFailure {
		t.Fatalf("expected exit code %d, got %d", exitIntegrityFailure, code)
	}
	if !strings.Contains(stdout.String(), "CRITICAL") {
		t.Fatalf("stdout missing CRITICAL, got: %q", stdout.String())
	}
}

// ── S3 error handling ─────────────────────────────────────────────────────────

func TestRunVerifyRemote404(t *testing.T) {
	remoteURI := "s3://my-bucket/prefix/sha256-" + strings.Repeat("a", 64) + ".atb"
	fake := &fakeS3Uploader{getErr: &push.S3Error{StatusCode: 404, Body: "NoSuchKey"}}
	setRemoteUploader(t, fake)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--remote", remoteURI}, &stdout, &stderr)

	if code != exitUserError {
		t.Fatalf("expected exit code %d (user error for 404), got %d", exitUserError, code)
	}
	if !strings.Contains(stderr.String(), "download") {
		t.Fatalf("stderr %q missing download error", stderr.String())
	}
}

func TestRunVerifyRemoteAuthError(t *testing.T) {
	remoteURI := "s3://my-bucket/prefix/sha256-" + strings.Repeat("a", 64) + ".atb"
	fake := &fakeS3Uploader{getErr: &push.S3Error{StatusCode: 403, Body: "AccessDenied"}}
	setRemoteUploader(t, fake)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--remote", remoteURI}, &stdout, &stderr)

	if code != exitSystemError {
		t.Fatalf("expected exit code %d (system error for 403), got %d", exitSystemError, code)
	}
}

// ── flag validation ───────────────────────────────────────────────────────────

func TestRunVerifyRemoteAndBundleMutuallyExclusive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runVerify([]string{
		"--remote", "s3://bucket/key",
		"--bundle", "some.atb",
	}, &stdout, &stderr)

	if code != exitUserError {
		t.Fatalf("expected exit code %d, got %d", exitUserError, code)
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("stderr %q missing 'mutually exclusive'", stderr.String())
	}
}

func TestRunVerifyRemoteMissingValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--remote"}, &stdout, &stderr)
	if code != exitUserError {
		t.Fatalf("expected exit code %d, got %d", exitUserError, code)
	}
}

// ── compile-time interface check ─────────────────────────────────────────────

// fakeS3Uploader already declared in push_test.go (same package).
// Verify it satisfies push.S3Uploader (redundant with push_test.go but explicit here).
var _ push.S3Uploader = (*fakeS3Uploader)(nil)
