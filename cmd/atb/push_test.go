// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/push"
)

// ── fake S3 uploader ─────────────────────────────────────────────────────────

type pushRoundTripFunc func(*http.Request) (*http.Response, error)

func (f pushRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type fakeS3Uploader struct {
	putCalls []*push.PutObjectInput
	getCalls []*push.GetObjectInput
	putErr   error
	getBody  []byte
	getErr   error
}

func (f *fakeS3Uploader) PutObject(_ context.Context, in push.PutObjectInput) (push.PutObjectOutput, error) {
	cp := in
	f.putCalls = append(f.putCalls, &cp)
	if f.putErr != nil {
		return push.PutObjectOutput{}, f.putErr
	}
	return push.PutObjectOutput{ETag: `"test-etag"`}, nil
}

func (f *fakeS3Uploader) GetObject(_ context.Context, in push.GetObjectInput) (push.GetObjectOutput, error) {
	cp := in
	f.getCalls = append(f.getCalls, &cp)
	if f.getErr != nil {
		return push.GetObjectOutput{}, f.getErr
	}
	body := f.getBody
	if body == nil {
		body = []byte{}
	}
	return push.GetObjectOutput{Body: newBytesReadCloser(body)}, nil
}

// bytesReadCloser wraps a byte slice as io.ReadCloser.
type bytesReadCloser struct{ *bytes.Reader }

func (b bytesReadCloser) Close() error { return nil }

func newBytesReadCloser(data []byte) bytesReadCloser {
	return bytesReadCloser{bytes.NewReader(data)}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// newTestBundleFile creates a temp bundle with one event and returns its path and head hash.
func newTestBundleFile(t *testing.T) (path, headHash string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "test.atb")
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "atb.snapshot", map[string]string{"name": "test"})
	if err := b.Save(path); err != nil {
		t.Fatalf("save test bundle: %v", err)
	}
	headHash = b.Records[len(b.Records)-1].Hash
	return path, headHash
}

// ── parsePushArgs tests ───────────────────────────────────────────────────────

func TestParsePushArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		want       pushConfig
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "target only",
			args: []string{"s3://my-bucket/atb"},
			want: pushConfig{
				Target:     "s3://my-bucket/atb",
				BundlePath: bundle.DefaultPath(),
				Format:     verifyFormatText,
			},
		},
		{
			name: "target with lock-until",
			args: []string{"s3://my-bucket/atb", "--lock-until", "2028-01-01"},
			want: pushConfig{
				Target:     "s3://my-bucket/atb",
				BundlePath: bundle.DefaultPath(),
				LockUntil:  "2028-01-01",
				Format:     verifyFormatText,
			},
		},
		{
			name: "lock-until equals syntax",
			args: []string{"s3://my-bucket/atb", "--lock-until=2029-06-30"},
			want: pushConfig{
				Target:     "s3://my-bucket/atb",
				BundlePath: bundle.DefaultPath(),
				LockUntil:  "2029-06-30",
				Format:     verifyFormatText,
			},
		},
		{
			name: "dry-run and json format",
			args: []string{"s3://my-bucket/atb", "--dry-run", "--format", "json"},
			want: pushConfig{
				Target:     "s3://my-bucket/atb",
				BundlePath: bundle.DefaultPath(),
				DryRun:     true,
				Format:     verifyFormatJSON,
			},
		},
		{
			name: "json format equals syntax",
			args: []string{"s3://my-bucket/atb", "--format=json"},
			want: pushConfig{
				Target:     "s3://my-bucket/atb",
				BundlePath: bundle.DefaultPath(),
				Format:     verifyFormatJSON,
			},
		},
		{
			name: "missing target",
			args: nil,
			want: pushConfig{
				BundlePath: bundle.DefaultPath(),
				Format:     verifyFormatText,
			},
		},
		{
			name:       "duplicate target",
			args:       []string{"s3://bucket-a/pfx", "s3://bucket-b/pfx"},
			wantErr:    true,
			wantErrMsg: "target already set",
		},
		{
			name:       "missing lock-until value",
			args:       []string{"s3://my-bucket/atb", "--lock-until"},
			wantErr:    true,
			wantErrMsg: "missing value for --lock-until",
		},
		{
			name:       "missing format value",
			args:       []string{"s3://my-bucket/atb", "--format"},
			wantErr:    true,
			wantErrMsg: "missing value for --format",
		},
		{
			name:       "invalid format",
			args:       []string{"s3://my-bucket/atb", "--format", "yaml"},
			wantErr:    true,
			wantErrMsg: "invalid format",
		},
		{
			name:       "unknown flag",
			args:       []string{"s3://my-bucket/atb", "--region", "us-east-1"},
			wantErr:    true,
			wantErrMsg: "unknown flag",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePushArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.wantErrMsg != "" && !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Target != tc.want.Target {
				t.Fatalf("Target: got %q want %q", got.Target, tc.want.Target)
			}
			if got.BundlePath != tc.want.BundlePath {
				t.Fatalf("BundlePath: got %q want %q", got.BundlePath, tc.want.BundlePath)
			}
			if got.LockUntil != tc.want.LockUntil {
				t.Fatalf("LockUntil: got %q want %q", got.LockUntil, tc.want.LockUntil)
			}
			if got.DryRun != tc.want.DryRun {
				t.Fatalf("DryRun: got %v want %v", got.DryRun, tc.want.DryRun)
			}
			if got.Format != tc.want.Format {
				t.Fatalf("Format: got %q want %q", got.Format, tc.want.Format)
			}
		})
	}
}

// ── runPush integration tests ─────────────────────────────────────────────────

func TestRunPushHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPush([]string{"--help"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d", exitSuccess, code)
	}
	if !strings.Contains(stdout.String(), "atb push") {
		t.Fatalf("stdout %q missing usage line", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--lock-until") {
		t.Fatalf("stdout %q missing --lock-until flag", stdout.String())
	}
}

func TestRunPushMissingTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPush(nil, &stdout, &stderr)
	if code != exitUserError {
		t.Fatalf("expected exit code %d, got %d", exitUserError, code)
	}
	if !strings.Contains(stderr.String(), "target URI required") {
		t.Fatalf("stderr %q missing expected message", stderr.String())
	}
}

func TestRunPushMissingTargetJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPush([]string{"--format", "json"}, &stdout, &stderr)
	if code != exitUserError {
		t.Fatalf("expected exit code %d, got %d", exitUserError, code)
	}
	if stdout.Len() == 0 {
		t.Fatalf("expected JSON error on stdout, got nothing")
	}
	var res pushResult
	if err := json.NewDecoder(&stdout).Decode(&res); err != nil {
		t.Fatalf("decode push result: %v", err)
	}
	if res.Status != "error" {
		t.Fatalf("status: got %q want %q", res.Status, "error")
	}
}

func TestRunPushMissingBundle(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPush([]string{"s3://my-bucket/atb", "--bundle", "/nonexistent/bundle.atb"}, &stdout, &stderr)
	// Missing bundle is a user/input error
	if code != exitUserError {
		t.Fatalf("expected exit code %d, got %d", exitUserError, code)
	}
	if !strings.Contains(stderr.String(), "bundle") {
		t.Fatalf("stderr %q missing bundle error", stderr.String())
	}
}

func TestRunPushMissingBundleJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPush([]string{"s3://my-bucket/atb", "--bundle", "/nonexistent/bundle.atb", "--format", "json"}, &stdout, &stderr)
	if code != exitUserError {
		t.Fatalf("expected exit code %d, got %d", exitUserError, code)
	}
	var res pushResult
	if err := json.NewDecoder(&stdout).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Status != "error" {
		t.Fatalf("status: got %q want error", res.Status)
	}
}

func TestRunPushDryRun(t *testing.T) {
	bundlePath, headHash := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath, "--dry-run",
	}, &stdout, &stderr, fake)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr=%q)", exitSuccess, code, stderr.String())
	}
	// No S3 calls made
	if len(fake.putCalls) != 0 {
		t.Fatalf("dry-run must not make S3 calls, got %d PutObject calls", len(fake.putCalls))
	}
	out := stdout.String()
	if !strings.Contains(out, headHash) {
		t.Fatalf("stdout %q missing head hash %q", out, headHash)
	}
	if !strings.Contains(out, "sha256-"+headHash+".atb") {
		t.Fatalf("stdout %q missing key filename", out)
	}
}

func TestRunPushDryRunJSON(t *testing.T) {
	bundlePath, headHash := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath, "--dry-run", "--format", "json",
	}, &stdout, &stderr, fake)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d", exitSuccess, code)
	}
	if len(fake.putCalls) != 0 {
		t.Fatalf("dry-run must not make S3 calls")
	}

	var res pushResult
	if err := json.NewDecoder(&stdout).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Status != "ok" {
		t.Fatalf("status: got %q want ok", res.Status)
	}
	if !res.DryRun {
		t.Fatalf("dry_run field should be true")
	}
	if res.BundleHash != headHash {
		t.Fatalf("bundle_hash: got %q want %q", res.BundleHash, headHash)
	}
	if !strings.HasSuffix(res.ObjectKey, "sha256-"+headHash+".atb") {
		t.Fatalf("object_key %q missing expected filename", res.ObjectKey)
	}
}

func TestRunPushSuccess(t *testing.T) {
	bundlePath, headHash := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb-prod", "--bundle", bundlePath,
	}, &stdout, &stderr, fake)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr=%q)", exitSuccess, code, stderr.String())
	}
	if len(fake.putCalls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(fake.putCalls))
	}
	call := fake.putCalls[0]
	if call.Bucket != "my-bucket" {
		t.Fatalf("bucket: got %q want my-bucket", call.Bucket)
	}
	wantKey := "atb-prod/sha256-" + headHash + ".atb"
	if call.Key != wantKey {
		t.Fatalf("key: got %q want %q", call.Key, wantKey)
	}
	if call.LockMode != "" {
		t.Fatalf("lock mode should be empty without --lock-until")
	}

	// Successful push must not mutate the local bundle.
	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("reload bundle: %v", err)
	}
	if got := len(b.Records); got != 2 {
		t.Fatalf("expected bundle to remain unchanged with 2 records, got %d", got)
	}
	last := b.Records[len(b.Records)-1]
	if last.Event.Type == "atb.bundle.pushed" {
		t.Fatalf("did not expect atb.bundle.pushed to be appended")
	}
	if last.Hash != headHash {
		t.Fatalf("last hash: got %q want %q", last.Hash, headHash)
	}
}

func TestRunPushSuccessJSON(t *testing.T) {
	bundlePath, headHash := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath, "--format", "json",
	}, &stdout, &stderr, fake)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d", exitSuccess, code)
	}

	var res pushResult
	if err := json.NewDecoder(&stdout).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Status != "ok" {
		t.Fatalf("status: got %q want ok", res.Status)
	}
	if res.Action != "push" {
		t.Fatalf("action: got %q want push", res.Action)
	}
	if res.BundleHash != headHash {
		t.Fatalf("bundle_hash: got %q want %q", res.BundleHash, headHash)
	}
	if !strings.HasSuffix(res.ObjectKey, "sha256-"+headHash+".atb") {
		t.Fatalf("object_key %q missing expected filename", res.ObjectKey)
	}
}

func TestRunPushWithLockUntil(t *testing.T) {
	bundlePath, _ := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath, "--lock-until", "2028-01-01",
	}, &stdout, &stderr, fake)

	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr=%q)", exitSuccess, code, stderr.String())
	}
	if len(fake.putCalls) != 1 {
		t.Fatalf("expected 1 PutObject call")
	}
	call := fake.putCalls[0]
	if call.LockMode != "COMPLIANCE" {
		t.Fatalf("lock mode: got %q want COMPLIANCE", call.LockMode)
	}
	if call.LockUntil != "2028-01-01T00:00:00Z" {
		t.Fatalf("lock until: got %q want 2028-01-01T00:00:00Z", call.LockUntil)
	}
}

func TestRunPushCredentialError(t *testing.T) {
	bundlePath, _ := newTestBundleFile(t)
	authErr := &push.S3Error{StatusCode: 403, Body: "AccessDenied"}
	fake := &fakeS3Uploader{putErr: authErr}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath,
	}, &stdout, &stderr, fake)

	if code != exitSystemError {
		t.Fatalf("expected exit code %d, got %d", exitSystemError, code)
	}
	if !strings.Contains(stderr.String(), "credential") {
		t.Fatalf("stderr %q missing credential error", stderr.String())
	}
}

func TestRunPushCredentialErrorJSON(t *testing.T) {
	bundlePath, _ := newTestBundleFile(t)
	authErr := &push.S3Error{StatusCode: 403, Body: "AccessDenied"}
	fake := &fakeS3Uploader{putErr: authErr}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath, "--format", "json",
	}, &stdout, &stderr, fake)

	if code != exitSystemError {
		t.Fatalf("expected exit code %d, got %d", exitSystemError, code)
	}
	var res pushResult
	if err := json.NewDecoder(&stdout).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Status != "error" {
		t.Fatalf("status: got %q want error", res.Status)
	}
	if !strings.Contains(res.Error, "credential") {
		t.Fatalf("error field %q missing credential", res.Error)
	}
}

func TestRunPushUploadError(t *testing.T) {
	bundlePath, _ := newTestBundleFile(t)
	fake := &fakeS3Uploader{putErr: errors.New("connection refused")}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath,
	}, &stdout, &stderr, fake)

	if code != exitSystemError {
		t.Fatalf("expected exit code %d, got %d", exitSystemError, code)
	}
}

func TestRunPushInvalidLockUntil(t *testing.T) {
	bundlePath, _ := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{
		"s3://my-bucket/atb", "--bundle", bundlePath, "--lock-until", "not-a-date",
	}, &stdout, &stderr, fake)

	if code != exitUserError {
		t.Fatalf("expected exit code %d, got %d", exitUserError, code)
	}
	if len(fake.putCalls) != 0 {
		t.Fatalf("no S3 call should be made on invalid lock-until")
	}
}

func TestRunPushKeyScheme(t *testing.T) {
	// Verify that the key construction matches the content-addressed scheme:
	// <prefix>/sha256-<head-hash>.atb
	bundlePath, headHash := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	runPushWithUploader([]string{
		"s3://bucket/my/prefix", "--bundle", bundlePath,
	}, &bytes.Buffer{}, &bytes.Buffer{}, fake)

	if len(fake.putCalls) != 1 {
		t.Fatalf("expected 1 PutObject call")
	}
	wantKey := "my/prefix/sha256-" + headHash + ".atb"
	if fake.putCalls[0].Key != wantKey {
		t.Fatalf("key: got %q want %q", fake.putCalls[0].Key, wantKey)
	}
}

func TestRunPushUsesConfigDefaults(t *testing.T) {
	bundlePath, headHash := newTestBundleFile(t)
	fake := &fakeS3Uploader{}

	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	cfgModel := atbConfig{
		Version: configVersion,
		Push: &pushSettings{
			Target:    "s3://cfg-bucket/cfg-prefix",
			LockMode:  "GOVERNANCE",
			LockUntil: "2029-01-01",
		},
	}
	if err := saveATBConfig(defaultConfigPath(), cfgModel); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{"--bundle", bundlePath}, &stdout, &stderr, fake)
	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr=%q)", exitSuccess, code, stderr.String())
	}
	if len(fake.putCalls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(fake.putCalls))
	}
	call := fake.putCalls[0]
	if call.Bucket != "cfg-bucket" {
		t.Fatalf("bucket: got %q want cfg-bucket", call.Bucket)
	}
	wantKey := "cfg-prefix/sha256-" + headHash + ".atb"
	if call.Key != wantKey {
		t.Fatalf("key: got %q want %q", call.Key, wantKey)
	}
	if call.LockMode != "GOVERNANCE" {
		t.Fatalf("lock mode: got %q want GOVERNANCE", call.LockMode)
	}
	if call.LockUntil != "2029-01-01T00:00:00Z" {
		t.Fatalf("lock until: got %q want 2029-01-01T00:00:00Z", call.LockUntil)
	}
}

func TestRunPushWithConfiguredHTTPEndpoint(t *testing.T) {
	bundlePath, headHash := newTestBundleFile(t)

	var gotMethod string
	var gotURL string
	var gotLockMode string
	var gotLockUntil string
	var gotBody []byte

	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	cfgModel := atbConfig{
		Version: configVersion,
		Push: &pushSettings{
			Target:            "s3://cfg-bucket/cfg-prefix",
			EndpointURL:       "http://storage.example.test",
			Region:            "ap-southeast-2",
			CredentialsSource: "aws_env_shared",
		},
	}
	if err := saveATBConfig(defaultConfigPath(), cfgModel); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")

	uploader, err := push.NewHTTPClientWithConfig(push.ClientConfig{
		EndpointURL: "http://storage.example.test",
		Region:      "ap-southeast-2",
		HTTPClient: &http.Client{
			Transport: pushRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				gotMethod = r.Method
				gotURL = r.URL.String()
				gotLockMode = r.Header.Get("x-amz-object-lock-mode")
				gotLockUntil = r.Header.Get("x-amz-object-lock-retain-until-date")
				body, err := io.ReadAll(r.Body)
				if err != nil {
					return nil, err
				}
				gotBody = body
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Etag": []string{`"etag-cli"`}},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    r,
				}, nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewHTTPClientWithConfig: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := runPushWithUploader([]string{"--bundle", bundlePath, "--lock-until", "2028-01-01"}, &stdout, &stderr, uploader)
	if code != exitSuccess {
		t.Fatalf("expected exit code %d, got %d (stderr=%q)", exitSuccess, code, stderr.String())
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method: got %q want %q", gotMethod, http.MethodPut)
	}
	if gotURL != "http://storage.example.test/cfg-bucket/cfg-prefix/sha256-"+headHash+".atb" {
		t.Fatalf("URL: got %q", gotURL)
	}
	if gotLockMode != "COMPLIANCE" {
		t.Fatalf("lock mode: got %q want COMPLIANCE", gotLockMode)
	}
	if gotLockUntil != "2028-01-01T00:00:00Z" {
		t.Fatalf("lock until: got %q want 2028-01-01T00:00:00Z", gotLockUntil)
	}
	wantBody, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle bytes: %v", err)
	}
	if string(gotBody) != string(wantBody) {
		t.Fatalf("uploaded body does not match local bundle bytes")
	}
	if !strings.Contains(stdout.String(), "s3://cfg-bucket/cfg-prefix/sha256-"+headHash+".atb") {
		t.Fatalf("stdout %q missing remote URI", stdout.String())
	}
}

// TestUsageJSONIncludesPush checks push appears in the machine-readable help.
func TestUsageJSONIncludesPush(t *testing.T) {
	found := false
	for _, cmd := range usageJSON().Commands {
		if cmd.Name == "push" {
			found = true
			if !strings.Contains(cmd.Usage, "--lock-until") {
				t.Fatalf("push usage missing --lock-until: %q", cmd.Usage)
			}
			if !strings.Contains(cmd.Usage, "s3://") {
				t.Fatalf("push usage missing s3:// example: %q", cmd.Usage)
			}
		}
	}
	if !found {
		t.Fatalf("push command missing from usageJSON()")
	}
}

// TestParseLockUntil validates date parsing and RFC 3339 conversion.
func TestParseLockUntil(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"2028-01-01", "2028-01-01T00:00:00Z", false},
		{"2099-12-31", "2099-12-31T00:00:00Z", false},
		{"not-a-date", "", true},
		{"2028/01/01", "", true},
		{"", "", true},
	}
	for _, tc := range tests {
		got, err := parseLockUntil(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseLockUntil(%q): expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseLockUntil(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseLockUntil(%q): got %q want %q", tc.input, got, tc.want)
		}
	}
}

// Ensure fakeS3Uploader satisfies the interface (compile-time check).
var _ push.S3Uploader = (*fakeS3Uploader)(nil)

// Ensure HTTPS3Client satisfies the interface (compile-time check).
var _ push.S3Uploader = (*push.HTTPS3Client)(nil)

// suppress unused import
var _ = os.DevNull
