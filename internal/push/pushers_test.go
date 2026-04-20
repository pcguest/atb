package push

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestS3PusherPush_SetsObjectLockHeaders(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")

	var gotMethod string
	var gotPath string
	var gotLockMode string
	var gotLockUntil string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotLockMode = r.Header.Get("x-amz-object-lock-mode")
		gotLockUntil = r.Header.Get("x-amz-object-lock-retain-until-date")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		gotBody = body

		w.Header().Set("ETag", `"etag-s3"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewHTTPClientWithConfig(ClientConfig{
		EndpointURL: server.URL,
		Region:      "ap-southeast-2",
	})
	if err != nil {
		t.Fatalf("NewHTTPClientWithConfig: %v", err)
	}

	pusher := S3Pusher{
		Uploader:  client,
		Bucket:    "audit-bucket",
		Key:       "exports/sha256-head.atb",
		LockMode:  "COMPLIANCE",
		LockUntil: "2028-01-01T00:00:00Z",
	}

	bundleBytes := []byte("bundle-bytes")
	if err := pusher.Push(context.Background(), bundleBytes, PushMeta{}); err != nil {
		t.Fatalf("Push: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method: got %q want %q", gotMethod, http.MethodPut)
	}
	if gotPath != "/audit-bucket/exports/sha256-head.atb" {
		t.Fatalf("path: got %q want %q", gotPath, "/audit-bucket/exports/sha256-head.atb")
	}
	if gotLockMode != "COMPLIANCE" {
		t.Fatalf("lock mode: got %q want %q", gotLockMode, "COMPLIANCE")
	}
	if gotLockUntil != "2028-01-01T00:00:00Z" {
		t.Fatalf("lock until: got %q want %q", gotLockUntil, "2028-01-01T00:00:00Z")
	}
	if string(gotBody) != string(bundleBytes) {
		t.Fatalf("body: got %q want %q", string(gotBody), string(bundleBytes))
	}
}

func TestQueuePusherPush_SendsEnvelopeAndSignature(t *testing.T) {
	key := []byte("queue-secret-key")
	meta := PushMeta{
		BundleID:      "bundle-1234",
		Digest:        strings.Repeat("a", 64),
		SealTimestamp: time.Date(2026, time.April, 20, 3, 4, 5, 0, time.UTC),
		ProfileID:     "atb.profile.privileged_tool_action",
	}

	var gotSignature string
	var gotEnvelope QueueEnvelope
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method: got %q want %q", r.Method, http.MethodPost)
		}
		gotSignature = r.Header.Get("X-ATB-Signature")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		gotBody = body

		if err := json.Unmarshal(body, &gotEnvelope); err != nil {
			t.Fatalf("unmarshal envelope: %v", err)
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	pusher := QueuePusher{
		EndpointURL: server.URL,
		HMACKey:     key,
		ATBVersion:  "1.9.0",
	}

	if err := pusher.Push(context.Background(), []byte("bundle-bytes"), meta); err != nil {
		t.Fatalf("Push: %v", err)
	}

	if gotEnvelope.BundleID != meta.BundleID {
		t.Fatalf("bundle_id: got %q want %q", gotEnvelope.BundleID, meta.BundleID)
	}
	if gotEnvelope.Digest != meta.Digest {
		t.Fatalf("digest: got %q want %q", gotEnvelope.Digest, meta.Digest)
	}
	if gotEnvelope.SealTimestamp != "2026-04-20T03:04:05Z" {
		t.Fatalf("seal_timestamp: got %q want %q", gotEnvelope.SealTimestamp, "2026-04-20T03:04:05Z")
	}
	if gotEnvelope.ProfileID != meta.ProfileID {
		t.Fatalf("profile_id: got %q want %q", gotEnvelope.ProfileID, meta.ProfileID)
	}
	if gotEnvelope.ATBVersion != "1.9.0" {
		t.Fatalf("atb_version: got %q want %q", gotEnvelope.ATBVersion, "1.9.0")
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(gotBody)
	wantSignature := hex.EncodeToString(mac.Sum(nil))
	if gotSignature != wantSignature {
		t.Fatalf("signature: got %q want %q", gotSignature, wantSignature)
	}
}
