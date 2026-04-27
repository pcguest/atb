// SPDX-License-Identifier: MIT
// Package push implements the S3 upload layer for atb push and atb verify --remote.
//
// The public interface is S3Uploader. The real implementation (HTTPS3Client) uses
// net/http with AWS Signature V4 signing. Tests inject a fake via the interface.
package push

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// S3Uploader is the minimal S3 interface needed by atb push and atb verify --remote.
type S3Uploader interface {
	PutObject(ctx context.Context, in PutObjectInput) (PutObjectOutput, error)
	GetObject(ctx context.Context, in GetObjectInput) (GetObjectOutput, error)
}

// PutObjectInput carries parameters for an S3 PutObject call.
type PutObjectInput struct {
	Bucket    string
	Key       string
	Body      []byte
	LockMode  string // "COMPLIANCE" or "" (no object-lock header)
	LockUntil string // RFC 3339 datetime e.g. "2028-01-01T00:00:00Z", or ""
}

// PutObjectOutput is the result of a successful PutObject call.
type PutObjectOutput struct {
	ETag string
}

// GetObjectInput carries parameters for an S3 GetObject call.
type GetObjectInput struct {
	Bucket string
	Key    string
}

// GetObjectOutput carries the response body. Callers must close Body.
type GetObjectOutput struct {
	Body io.ReadCloser
}

// S3Error is returned for non-2xx S3 responses.
type S3Error struct {
	StatusCode int
	Body       string
}

func (e *S3Error) Error() string {
	return fmt.Sprintf("S3 HTTP %d: %s", e.StatusCode, strings.TrimSpace(e.Body))
}

// IsAuthError reports whether err is an S3 credential/permission error (401 or 403).
func IsAuthError(err error) bool {
	var s3err *S3Error
	if asS3Error(err, &s3err) {
		return s3err.StatusCode == 401 || s3err.StatusCode == 403
	}
	return false
}

// IsNotFound reports whether err is an S3 404 response.
func IsNotFound(err error) bool {
	var s3err *S3Error
	if asS3Error(err, &s3err) {
		return s3err.StatusCode == 404
	}
	return false
}

// asS3Error is a minimal errors.As substitute that avoids an import cycle.
func asS3Error(err error, target **S3Error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*S3Error); ok {
		*target = e
		return true
	}
	// Unwrap one level (covers fmt.Errorf("%w", ...) wrappers).
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok {
		return asS3Error(u.Unwrap(), target)
	}
	return false
}

// ParseS3URI splits an S3 URI such as "s3://bucket/prefix" into bucket and prefix.
// The prefix has no leading or trailing slash. Bucket-only URIs ("s3://bucket") are
// valid and return an empty prefix.
func ParseS3URI(uri string) (bucket, prefix string, err error) {
	if !strings.HasPrefix(uri, "s3://") {
		return "", "", fmt.Errorf("not an S3 URI: %q (expected s3://bucket[/prefix])", uri)
	}
	rest := strings.TrimPrefix(uri, "s3://")
	if rest == "" {
		return "", "", fmt.Errorf("S3 URI missing bucket: %q", uri)
	}
	idx := strings.Index(rest, "/")
	if idx == -1 {
		return rest, "", nil
	}
	bucket = rest[:idx]
	prefix = strings.Trim(rest[idx+1:], "/")
	if bucket == "" {
		return "", "", fmt.Errorf("S3 URI missing bucket: %q", uri)
	}
	return bucket, prefix, nil
}

// ObjectKey builds the S3 object key from a prefix and bundle head hash.
//
//	prefix="" → "sha256-<hash>.atb"
//	prefix="atb/prod" → "atb/prod/sha256-<hash>.atb"
func ObjectKey(prefix, headHash string) string {
	filename := "sha256-" + headHash + ".atb"
	if prefix == "" {
		return filename
	}
	return strings.TrimSuffix(prefix, "/") + "/" + filename
}

// KeyHash extracts the expected head hash from an S3 object key or full S3 URI.
// The key must contain a filename component matching "sha256-<hex>.atb".
// Returns ("", false) if the pattern is not present.
func KeyHash(keyOrURI string) (string, bool) {
	// Strip scheme+bucket from a full URI: s3://bucket/rest → rest
	if strings.HasPrefix(keyOrURI, "s3://") {
		withoutScheme := keyOrURI[5:]
		idx := strings.Index(withoutScheme, "/")
		if idx == -1 {
			return "", false
		}
		keyOrURI = withoutScheme[idx+1:]
	}

	// Extract filename (last path segment)
	base := keyOrURI
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	if !strings.HasPrefix(base, "sha256-") || !strings.HasSuffix(base, ".atb") {
		return "", false
	}
	h := strings.TrimPrefix(base, "sha256-")
	h = strings.TrimSuffix(h, ".atb")
	if h == "" {
		return "", false
	}
	return h, true
}
