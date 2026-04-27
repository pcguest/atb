// SPDX-License-Identifier: MIT
package push

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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
