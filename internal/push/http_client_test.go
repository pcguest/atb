// SPDX-License-Identifier: MIT
package push

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type closeErrorBody struct {
	io.Reader
	err error
}

func (b closeErrorBody) Close() error {
	return b.err
}

func TestHTTPS3ClientPutObject_CustomEndpoint(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")

	var gotMethod string
	var gotURL string
	var gotContentType string
	var gotLockMode string
	var gotLockUntil string
	var gotAuth string
	var gotBody string

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotMethod = r.Method
			gotURL = r.URL.String()
			gotContentType = r.Header.Get("Content-Type")
			gotLockMode = r.Header.Get("x-amz-object-lock-mode")
			gotLockUntil = r.Header.Get("x-amz-object-lock-retain-until-date")
			gotAuth = r.Header.Get("Authorization")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			gotBody = string(body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Etag": []string{`"etag-1"`}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    r,
			}, nil
		}),
	}

	client, err := NewHTTPClientWithConfig(ClientConfig{
		EndpointURL: "http://storage.example.test",
		Region:      "ap-southeast-2",
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatalf("NewHTTPClientWithConfig: %v", err)
	}

	out, err := client.PutObject(context.Background(), PutObjectInput{
		Bucket:    "audit-bucket",
		Key:       "exports/sha256-abc123.atb",
		Body:      []byte("bundle-bytes"),
		LockMode:  "COMPLIANCE",
		LockUntil: "2028-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if out.ETag != `"etag-1"` {
		t.Fatalf("ETag: got %q want %q", out.ETag, `"etag-1"`)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method: got %q want %q", gotMethod, http.MethodPut)
	}
	if gotURL != "http://storage.example.test/audit-bucket/exports/sha256-abc123.atb" {
		t.Fatalf("URL: got %q", gotURL)
	}
	if gotContentType != "application/octet-stream" {
		t.Fatalf("content-type: got %q want application/octet-stream", gotContentType)
	}
	if gotLockMode != "COMPLIANCE" {
		t.Fatalf("lock mode: got %q want COMPLIANCE", gotLockMode)
	}
	if gotLockUntil != "2028-01-01T00:00:00Z" {
		t.Fatalf("lock until: got %q want 2028-01-01T00:00:00Z", gotLockUntil)
	}
	if gotBody != "bundle-bytes" {
		t.Fatalf("body: got %q want bundle-bytes", gotBody)
	}
	if !strings.Contains(gotAuth, "AWS4-HMAC-SHA256") {
		t.Fatalf("authorization header missing SigV4 marker: %q", gotAuth)
	}
}

func TestHTTPS3ClientPutObject_Non2xxReturnsS3Error(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("AccessDenied")),
				Request:    r,
			}, nil
		}),
	}

	client, err := NewHTTPClientWithConfig(ClientConfig{
		EndpointURL: "http://storage.example.test",
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatalf("NewHTTPClientWithConfig: %v", err)
	}

	_, err = client.PutObject(context.Background(), PutObjectInput{
		Bucket: "audit-bucket",
		Key:    "exports/sha256-abc123.atb",
		Body:   []byte("bundle-bytes"),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsAuthError(err) {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestNewHTTPClientWithConfig_UnsupportedCredentialsSource(t *testing.T) {
	_, err := NewHTTPClientWithConfig(ClientConfig{CredentialsSource: "oidc"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported credentials source") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPS3ClientGetObjectAndNotFound(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "session-token")

	requests := 0
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			if r.Method != http.MethodGet {
				t.Fatalf("method = %q, want GET", r.Method)
			}
			if !strings.Contains(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256") {
				t.Fatalf("missing SigV4 Authorization: %q", r.Header.Get("Authorization"))
			}
			if r.Header.Get("x-amz-security-token") != "session-token" {
				t.Fatalf("session token = %q", r.Header.Get("x-amz-security-token"))
			}
			status := http.StatusOK
			body := "bundle"
			if strings.Contains(r.URL.Path, "missing") {
				status = http.StatusNotFound
				body = "NoSuchKey"
			}
			return &http.Response{
				StatusCode: status,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	client, err := NewHTTPClientWithConfig(ClientConfig{
		EndpointURL: "https://storage.example.test/base/",
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatalf("NewHTTPClientWithConfig: %v", err)
	}
	out, err := client.GetObject(context.Background(), GetObjectInput{Bucket: "bucket", Key: "/path/bundle.atb"})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	data, err := io.ReadAll(out.Body)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	out.Body.Close()
	if string(data) != "bundle" {
		t.Fatalf("body = %q", data)
	}

	_, err = client.GetObject(context.Background(), GetObjectInput{Bucket: "bucket", Key: "missing"})
	if err == nil || !IsNotFound(err) || IsAuthError(err) {
		t.Fatalf("missing object error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestHTTPS3ClientGetObjectReportsNon2xxBodyCloseFailure(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	closeErr := errors.New("close failed")
	client, err := NewHTTPClientWithConfig(ClientConfig{
		EndpointURL: "https://storage.example.test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{},
				Body:       closeErrorBody{Reader: strings.NewReader("NoSuchKey"), err: closeErr},
				Request:    r,
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPClientWithConfig: %v", err)
	}
	_, err = client.GetObject(context.Background(), GetObjectInput{Bucket: "bucket", Key: "missing"})
	if err == nil || !IsNotFound(err) || !errors.Is(err, closeErr) {
		t.Fatalf("GetObject error=%v, want S3 not-found wrapping close failure", err)
	}
}

func TestHTTPS3ClientTransportErrors(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	transportErr := errors.New("transport unavailable")
	client, err := NewHTTPClientWithConfig(ClientConfig{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, transportErr
		})},
	})
	if err != nil {
		t.Fatalf("NewHTTPClientWithConfig: %v", err)
	}
	if _, err := client.PutObject(context.Background(), PutObjectInput{Bucket: "bucket", Key: "key"}); !errors.Is(err, transportErr) {
		t.Fatalf("PutObject error = %v", err)
	}
	if _, err := client.GetObject(context.Background(), GetObjectInput{Bucket: "bucket", Key: "key"}); !errors.Is(err, transportErr) {
		t.Fatalf("GetObject error = %v", err)
	}
}

func TestCredentialEndpointAndRegionResolution(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("HOME", t.TempDir())
	if _, err := ResolveCredentials(); err == nil {
		t.Fatal("ResolveCredentials unexpectedly succeeded without credentials")
	}

	for _, endpoint := range []string{"ftp://example.com", "https://", "relative/path"} {
		if _, err := normalizeEndpointURL(endpoint); err == nil {
			t.Errorf("normalizeEndpointURL(%q) unexpectedly succeeded", endpoint)
		}
	}
	if got, err := normalizeEndpointURL(" https://example.com/base/ "); err != nil || got != "https://example.com/base" {
		t.Fatalf("normalize endpoint = %q, %v", got, err)
	}
	if got, err := normalizeEndpointURL(""); err != nil || got != "" {
		t.Fatalf("normalize empty endpoint = %q, %v", got, err)
	}

	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_REGION", "")
	if got := ResolveRegion(); got != "us-east-1" {
		t.Fatalf("default region = %q", got)
	}
	t.Setenv("AWS_REGION", "ap-southeast-1")
	if got := ResolveRegion(); got != "ap-southeast-1" {
		t.Fatalf("environment region = %q", got)
	}
	if got := resolveRegion(" eu-west-1 "); got != "eu-west-1" {
		t.Fatalf("explicit region = %q", got)
	}
}

func TestSharedCredentialProfileResolution(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_PROFILE", "audit")
	awsDir := filepath.Join(home, ".aws")
	if err := os.MkdirAll(awsDir, 0o700); err != nil {
		t.Fatalf("mkdir .aws: %v", err)
	}
	content := strings.Join([]string{
		"# ignored",
		"[default]",
		"aws_access_key_id = default",
		"[audit]",
		"invalid-line",
		"aws_access_key_id = audit-access",
		"aws_secret_access_key = audit-secret",
		"aws_session_token = audit-session",
	}, "\n")
	if err := os.WriteFile(filepath.Join(awsDir, "credentials"), []byte(content), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	got, err := ResolveCredentials()
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}
	if got.AccessKeyID != "audit-access" || got.SecretAccessKey != "audit-secret" || got.SessionToken != "audit-session" {
		t.Fatalf("credentials = %+v", got)
	}

	t.Setenv("AWS_PROFILE", "missing")
	if _, err := ResolveCredentials(); err == nil {
		t.Fatal("missing profile unexpectedly resolved")
	}
}

func TestS3ErrorClassification(t *testing.T) {
	if got := (&S3Error{StatusCode: http.StatusForbidden, Body: " denied "}).Error(); got != "S3 HTTP 403: denied" {
		t.Fatalf("error string = %q", got)
	}
	if IsAuthError(errors.New("other")) || IsNotFound(errors.New("other")) {
		t.Fatal("ordinary error classified as S3 status")
	}
	if !IsAuthError(&S3Error{StatusCode: http.StatusUnauthorized}) {
		t.Fatal("401 not classified as auth error")
	}
}
