// SPDX-License-Identifier: MIT
package apiv1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/locator"
)

func locateRequest(t *testing.T, handler http.Handler, locatorValue string) (*httptest.ResponseRecorder, LocateResponse) {
	t.Helper()
	target := "/api/v1/bundle/locate"
	if locatorValue != "" {
		target += "?locator=" + url.QueryEscape(locatorValue)
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	var out LocateResponse
	if rr.Body.Len() > 0 {
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode LocateResponse: %v (body=%s)", err, rr.Body.String())
		}
	}
	return rr, out
}

func TestHandleBundleLocate(t *testing.T) {
	b := newTestBundle(t)
	appendTestBundleEvent(t, b, "test.one", map[string]any{"n": 1})
	appendTestBundleEvent(t, b, "test.two", map[string]any{"n": 2})
	_, handler := buildTestAPIServer(t, APIConfig{Bundle: b})

	head := b.Records[len(b.Records)-1].Hash
	target := b.Records[1]

	t.Run("resolves matching locator", func(t *testing.T) {
		loc := locator.Locator{BundleHeadHash: head, EventSequence: target.Event.Sequence}
		rr, out := locateRequest(t, handler, loc.String())
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 body=%s", rr.Code, rr.Body.String())
		}
		if !out.OK || out.Seq != target.Event.Sequence || out.RecordHashMatched {
			t.Fatalf("response = %+v, want ok seq=%d no hash match", out, target.Event.Sequence)
		}
		if out.Canonical != loc.String() {
			t.Fatalf("canonical = %q, want %q", out.Canonical, loc.String())
		}
	})

	t.Run("record hash match", func(t *testing.T) {
		loc := locator.Locator{BundleHeadHash: head, EventSequence: target.Event.Sequence, RecordHash: target.Hash}
		_, out := locateRequest(t, handler, loc.String())
		if !out.OK || !out.RecordHashMatched {
			t.Fatalf("response = %+v, want ok with record hash matched", out)
		}
	})

	t.Run("malformed locator", func(t *testing.T) {
		rr, out := locateRequest(t, handler, "atb://evidence/1/nope?seq=1")
		if rr.Code != http.StatusOK || out.OK {
			t.Fatalf("status=%d out=%+v, want 200 ok=false", rr.Code, out)
		}
		if out.ErrorCode != "LOCATOR_MALFORMED" {
			t.Fatalf("error_code = %q, want LOCATOR_MALFORMED", out.ErrorCode)
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		loc := locator.Locator{BundleHeadHash: head, EventSequence: target.Event.Sequence}
		bad := strings.Replace(loc.String(), "/1/", "/2/", 1)
		_, out := locateRequest(t, handler, bad)
		if out.OK || out.ErrorCode != "LOCATOR_VERSION_UNSUPPORTED" {
			t.Fatalf("out = %+v, want LOCATOR_VERSION_UNSUPPORTED", out)
		}
	})

	t.Run("wrong bundle head", func(t *testing.T) {
		loc := locator.Locator{BundleHeadHash: strings.Repeat("c", 64), EventSequence: target.Event.Sequence}
		_, out := locateRequest(t, handler, loc.String())
		if out.OK || out.ErrorCode != "BUNDLE_NOT_AVAILABLE" {
			t.Fatalf("out = %+v, want BUNDLE_NOT_AVAILABLE", out)
		}
	})

	t.Run("event not found", func(t *testing.T) {
		loc := locator.Locator{BundleHeadHash: head, EventSequence: 9999}
		_, out := locateRequest(t, handler, loc.String())
		if out.OK || out.ErrorCode != "EVENT_NOT_FOUND" {
			t.Fatalf("out = %+v, want EVENT_NOT_FOUND", out)
		}
	})

	t.Run("record hash mismatch", func(t *testing.T) {
		loc := locator.Locator{BundleHeadHash: head, EventSequence: target.Event.Sequence, RecordHash: strings.Repeat("d", 64)}
		_, out := locateRequest(t, handler, loc.String())
		if out.OK || out.ErrorCode != "RECORD_HASH_MISMATCH" {
			t.Fatalf("out = %+v, want RECORD_HASH_MISMATCH", out)
		}
	})

	t.Run("missing locator parameter", func(t *testing.T) {
		rr, _ := locateRequest(t, handler, "")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		srv := NewAPIServer(APIConfig{Bundle: b})
		mux := http.NewServeMux()
		srv.Register(mux)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/locate?locator="+url.QueryEscape("atb://evidence/1/"+head+"?seq=1"), nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}
	})
}
