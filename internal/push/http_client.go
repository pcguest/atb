package push

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HTTPS3Client is the real S3 implementation using net/http + AWS Signature V4.
// Create with NewHTTPClient; tests inject a fake via the S3Uploader interface.
type HTTPS3Client struct {
	region      string
	creds       AWSCredentials
	hc          *http.Client
	endpointURL string
}

// AWSCredentials holds the key material needed for SigV4 signing.
type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string // empty when not using temporary credentials
}

// ClientConfig controls how the HTTP S3 client is resolved.
type ClientConfig struct {
	EndpointURL       string
	Region            string
	CredentialsSource string
	HTTPClient        *http.Client
}

// NewHTTPClient creates an HTTPS3Client using the standard AWS credential chain.
// It resolves credentials and region from environment variables and
// ~/.aws/credentials but does not contact any network endpoint.
func NewHTTPClient() (*HTTPS3Client, error) {
	return NewHTTPClientWithConfig(ClientConfig{})
}

// NewHTTPClientWithConfig creates an HTTPS3Client with optional endpoint and
// region overrides for S3-compatible targets. Credential resolution remains
// limited to the existing environment and shared credentials sources.
func NewHTTPClientWithConfig(cfg ClientConfig) (*HTTPS3Client, error) {
	if err := validateCredentialsSource(cfg.CredentialsSource); err != nil {
		return nil, err
	}
	creds, err := ResolveCredentials()
	if err != nil {
		return nil, err
	}
	endpointURL, err := normalizeEndpointURL(cfg.EndpointURL)
	if err != nil {
		return nil, err
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 120 * time.Second}
	}
	return &HTTPS3Client{
		region:      resolveRegion(cfg.Region),
		creds:       creds,
		hc:          hc,
		endpointURL: endpointURL,
	}, nil
}

// virtualHostedEndpoint returns the virtual-hosted-style base URL for the bucket.
func (c *HTTPS3Client) virtualHostedEndpoint(bucket string) string {
	return "https://" + bucket + ".s3." + c.region + ".amazonaws.com"
}

func (c *HTTPS3Client) objectURL(bucket, key string) (string, error) {
	if c.endpointURL == "" {
		return c.virtualHostedEndpoint(bucket) + "/" + strings.TrimPrefix(key, "/"), nil
	}

	base, err := url.Parse(c.endpointURL)
	if err != nil {
		return "", fmt.Errorf("s3 endpoint: parse URL: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + bucket + "/" + strings.TrimPrefix(key, "/")
	return base.String(), nil
}

// PutObject uploads body to s3://bucket/key with optional Object Lock headers.
func (c *HTTPS3Client) PutObject(ctx context.Context, in PutObjectInput) (PutObjectOutput, error) {
	rawURL, err := c.objectURL(in.Bucket, in.Key)
	if err != nil {
		return PutObjectOutput{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, bytes.NewReader(in.Body))
	if err != nil {
		return PutObjectOutput{}, fmt.Errorf("s3 put-object: build request: %w", err)
	}
	req.ContentLength = int64(len(in.Body))
	req.Header.Set("Content-Type", "application/octet-stream")
	if in.LockMode != "" {
		req.Header.Set("x-amz-object-lock-mode", in.LockMode)
	}
	if in.LockUntil != "" {
		req.Header.Set("x-amz-object-lock-retain-until-date", in.LockUntil)
	}

	signRequest(req, in.Body, c.creds, c.region, time.Now().UTC())

	resp, err := c.hc.Do(req)
	if err != nil {
		return PutObjectOutput{}, fmt.Errorf("s3 put-object: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return PutObjectOutput{}, &S3Error{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return PutObjectOutput{ETag: resp.Header.Get("ETag")}, nil
}

// GetObject retrieves s3://bucket/key and returns the response body stream.
// Callers must close GetObjectOutput.Body.
func (c *HTTPS3Client) GetObject(ctx context.Context, in GetObjectInput) (GetObjectOutput, error) {
	rawURL, err := c.objectURL(in.Bucket, in.Key)
	if err != nil {
		return GetObjectOutput{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return GetObjectOutput{}, fmt.Errorf("s3 get-object: build request: %w", err)
	}

	signRequest(req, nil, c.creds, c.region, time.Now().UTC())

	resp, err := c.hc.Do(req)
	if err != nil {
		return GetObjectOutput{}, fmt.Errorf("s3 get-object: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return GetObjectOutput{}, &S3Error{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return GetObjectOutput{Body: resp.Body}, nil
}

// ── AWS Signature V4 ─────────────────────────────────────────────────────────

// signRequest adds AWS Signature V4 headers to req.
// body is the full request payload (nil or empty for GET).
// now must be UTC.
func signRequest(req *http.Request, body []byte, creds AWSCredentials, region string, now time.Time) {
	const service = "s3"
	datetime := now.Format("20060102T150405Z")
	date := now.Format("20060102")

	payloadHash := sha256Hex(body)

	// Mandatory SigV4 headers that must be set before building the canonical form.
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if creds.SessionToken != "" {
		req.Header.Set("x-amz-security-token", creds.SessionToken)
	}

	// Build the header map that will be signed.  Keys are lowercase.
	hmap := map[string]string{
		"host":                 req.URL.Host,
		"x-amz-date":           datetime,
		"x-amz-content-sha256": payloadHash,
	}
	if req.ContentLength > 0 {
		hmap["content-length"] = strconv.FormatInt(req.ContentLength, 10)
	}
	if ct := req.Header.Get("Content-Type"); ct != "" {
		hmap["content-type"] = ct
	}
	for _, h := range []string{
		"x-amz-object-lock-mode",
		"x-amz-object-lock-retain-until-date",
		"x-amz-security-token",
	} {
		if v := req.Header.Get(h); v != "" {
			hmap[h] = v
		}
	}

	names := make([]string, 0, len(hmap))
	for k := range hmap {
		names = append(names, k)
	}
	sort.Strings(names)

	var cb strings.Builder
	for _, n := range names {
		cb.WriteString(n)
		cb.WriteByte(':')
		cb.WriteString(strings.TrimSpace(hmap[n]))
		cb.WriteByte('\n')
	}
	canonicalHeaders := cb.String()
	signedHeaders := strings.Join(names, ";")

	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := date + "/" + region + "/" + service + "/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + datetime + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))

	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256(
				hmacSHA256([]byte("AWS4"+creds.SecretAccessKey), []byte(date)),
				[]byte(region),
			),
			[]byte(service),
		),
		[]byte("aws4_request"),
	)

	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential="+creds.AccessKeyID+"/"+scope+
			", SignedHeaders="+signedHeaders+
			", Signature="+signature)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// ── Credential resolution ─────────────────────────────────────────────────────

// ResolveCredentials resolves AWS credentials from:
//  1. Environment variables: AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY
//  2. Shared credentials file: ~/.aws/credentials (profile from AWS_PROFILE or "default")
//
// Returns an error if no credentials are found.
func ResolveCredentials() (AWSCredentials, error) {
	if id := os.Getenv("AWS_ACCESS_KEY_ID"); id != "" {
		return AWSCredentials{
			AccessKeyID:     id,
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		}, nil
	}
	creds, err := readSharedCredentials()
	if err == nil {
		return creds, nil
	}
	return AWSCredentials{}, fmt.Errorf(
		"no AWS credentials found; set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, " +
			"or configure ~/.aws/credentials")
}

// ResolveRegion returns the AWS region from standard env vars, defaulting to "us-east-1".
func ResolveRegion() string {
	return resolveRegion("")
}

func resolveRegion(explicit string) string {
	if r := strings.TrimSpace(explicit); r != "" {
		return r
	}
	for _, env := range []string{"AWS_DEFAULT_REGION", "AWS_REGION"} {
		if r := os.Getenv(env); r != "" {
			return r
		}
	}
	return "us-east-1"
}

func validateCredentialsSource(source string) error {
	switch strings.TrimSpace(source) {
	case "", "aws_env_shared":
		return nil
	default:
		return fmt.Errorf("unsupported credentials source %q", source)
	}
}

func normalizeEndpointURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint URL %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid endpoint URL %q: expected http or https", raw)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid endpoint URL %q: missing host", raw)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

// readSharedCredentials parses ~/.aws/credentials for the active profile.
func readSharedCredentials() (AWSCredentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return AWSCredentials{}, fmt.Errorf("credentials: home dir: %w", err)
	}

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	path := filepath.Join(home, ".aws", "credentials")
	f, err := os.Open(filepath.Clean(path)) // #nosec G304 -- operator-controlled path under ~
	if err != nil {
		return AWSCredentials{}, fmt.Errorf("credentials: open %s: %w", path, err)
	}
	defer f.Close()

	var creds AWSCredentials
	inProfile := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inProfile = line[1:len(line)-1] == profile
			continue
		}
		if !inProfile {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "aws_access_key_id":
			creds.AccessKeyID = strings.TrimSpace(kv[1])
		case "aws_secret_access_key":
			creds.SecretAccessKey = strings.TrimSpace(kv[1])
		case "aws_session_token":
			creds.SessionToken = strings.TrimSpace(kv[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return AWSCredentials{}, fmt.Errorf("credentials: scan %s: %w", path, err)
	}
	if creds.AccessKeyID == "" {
		return AWSCredentials{}, fmt.Errorf("credentials: profile %q not found in %s", profile, path)
	}
	return creds, nil
}
