// SPDX-License-Identifier: MIT
package proxy_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/identity"
	"github.com/pcguest/atb/internal/proxy"
)

// testRecorder is a mock BundleRecorder for testing purposes.
type testRecorder struct {
	*proxy.BundleRecorder
	AppendSessionCloseFunc func(*proxy.Session) error
}

func (tr *testRecorder) AppendSessionClose(sess *proxy.Session) error {
	if tr.AppendSessionCloseFunc != nil {
		return tr.AppendSessionCloseFunc(sess)
	}
	return tr.BundleRecorder.AppendSessionClose(sess)
}

func TestProxyConfigValidate(t *testing.T) {
	t.Parallel()

	valid := proxy.ProxyConfig{
		ListenAddr:  "127.0.0.1:8080",
		BundlePath:  "run.atb/bundle.atb",
		TargetHosts: []string{"api.openai.com"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	cases := []struct {
		name string
		cfg  proxy.ProxyConfig
	}{
		{
			name: "missing listen",
			cfg: proxy.ProxyConfig{
				BundlePath:  "run.atb/bundle.atb",
				TargetHosts: []string{"api.openai.com"},
			},
		},
		{
			name: "missing bundle",
			cfg: proxy.ProxyConfig{
				ListenAddr:  "127.0.0.1:8080",
				TargetHosts: []string{"api.openai.com"},
			},
		},
		{
			name: "missing targets",
			cfg: proxy.ProxyConfig{
				ListenAddr: "127.0.0.1:8080",
				BundlePath: "run.atb/bundle.atb",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRequestRecordToEvent(t *testing.T) {
	t.Parallel()

	rec := proxy.RequestRecord{
		SessionID:   "sess-1",
		Host:        "api.openai.com",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		Model:       "gpt-4.1",
		DisplayName: "Paddy Guest",
		Body:        []byte(`{"model":"gpt-4.1"}`),
		RecordedAt:  time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC),
	}
	ev, err := rec.ToEvent()
	if err != nil {
		t.Fatalf("ToEvent: %v", err)
	}
	if ev.Type != proxy.TypeLLMRequest {
		t.Fatalf("type = %q, want %q", ev.Type, proxy.TypeLLMRequest)
	}
	data, ok := ev.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type %T", ev.Data)
	}
	if data["session_id"] != "sess-1" {
		t.Fatalf("session_id = %v", data["session_id"])
	}
	actor, ok := data["actor"].(map[string]string)
	if !ok || actor["display_name"] != "Paddy Guest" {
		t.Fatalf("actor = %#v", data["actor"])
	}
}

func TestResolveIdentity(t *testing.T) {
	t.Parallel()

	cfg := proxy.ProxyConfig{
		ListenAddr:  "127.0.0.1:8080",
		BundlePath:  "run.atb/bundle.atb",
		TargetHosts: []string{"api.openai.com"},
		IdentityMap: map[string]string{
			"sk-test": "Paddy Guest",
		},
	}
	if got := cfg.ResolveIdentity("sk-test").DisplayName; got != "Paddy Guest" {
		t.Fatalf("ResolveIdentity = %q", got)
	}
	if got := cfg.ResolveIdentity("sk-other").DisplayName; got != "api-key:ther" {
		t.Fatalf("unexpected identity %q", got)
	}
}

func TestProxyHandleRequestUsesIdentityMap(t *testing.T) {
	t.Parallel()

	p, err := proxy.NewProxy(proxy.ProxyConfig{
		ListenAddr:  "127.0.0.1:8080",
		BundlePath:  "run.atb/bundle.atb",
		TargetHosts: proxy.DefaultTargetHosts("openai"),
		IdentityMap: map[string]string{"sk-test": "Paddy Guest"},
	}, nil, nil)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}

	rec := proxy.RequestRecord{
		SessionID: "sess-2",
		Host:      "api.openai.com",
		Method:    "POST",
		Path:      "/v1/chat/completions",
		APIKey:    "sk-test",
	}
	if err := p.HandleRequest(context.Background(), rec); err != nil {
		t.Fatalf("HandleRequest: %v", err)
	}
	if name := p.Config().ResolveIdentity("sk-test").DisplayName; name != "Paddy Guest" {
		t.Fatalf("identity = %q", name)
	}
}

func TestDefaultTargetHosts(t *testing.T) {
	t.Parallel()

	got := proxy.DefaultTargetHosts("openai", "anthropic", "custom.example")
	want := "api.openai.com,api.anthropic.com,custom.example"
	if strings.Join(got, ",") != want {
		t.Fatalf("hosts = %v, want %v", got, want)
	}
}

func TestExchangeCompleteEvent(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	recorder := proxy.NewBundleRecorder(bundlePath, nil)

	cfg := proxy.ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  bundlePath,
		TargetHosts: []string{"api.openai.com"},
		IdentityMap: map[string]string{
			"sk-test": "test-actor-id",
		},
		Identity: identity.DefaultChain(),
	}

	// Create a single proxy instance wired to the test recorder via the
	// test-only seam in export_test.go.
	p, err := proxy.NewProxyForTest(cfg, recorder)
	if err != nil {
		t.Fatalf("NewProxyForTest: %v", err)
	}

	// Simulate first exchange
	req1, _ := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-3.5-turbo"}`))
	req1.Header.Set("X-Api-Key", "sk-test")
	resp1 := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{"usage":{"prompt_tokens":10,"completion_tokens":20}}`)),
		Header:     make(http.Header),
	}
	if err := p.CaptureRequestForTest("api.openai.com", req1, []byte(`{"model":"gpt-3.5-turbo"}`)); err != nil {
		t.Fatalf("CaptureRequest 1 failed: %v", err)
	}
	if err := p.CaptureResponseForTest("api.openai.com", req1, resp1, []byte(`{"usage":{"prompt_tokens":10,"completion_tokens":20}}`)); err != nil {
		t.Fatalf("CaptureResponse 1 failed: %v", err)
	}

	// Simulate second exchange
	req2, _ := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-4"}`))
	req2.Header.Set("X-Api-Key", "sk-test")
	resp2 := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{"usage":{"prompt_tokens":15,"completion_tokens":25}}`)),
		Header:     make(http.Header),
	}
	if err := p.CaptureRequestForTest("api.openai.com", req2, []byte(`{"model":"gpt-4"}`)); err != nil {
		t.Fatalf("CaptureRequest 2 failed: %v", err)
	}
	if err := p.CaptureResponseForTest("api.openai.com", req2, resp2, []byte(`{"usage":{"prompt_tokens":15,"completion_tokens":25}}`)); err != nil {
		t.Fatalf("CaptureResponse 2 failed: %v", err)
	}

	// Explicitly stop the proxy to trigger session close
	if err := p.Stop(); err != nil {
		t.Fatalf("proxy.Stop failed: %v", err)
	}

	loaded, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}

	exchangeEvents := []map[string]any{}
	requestEvents := []map[string]any{}
	for _, rec := range loaded.Records {
		if rec.Event.Type == "atb.exchange.complete" {
			if data, ok := rec.Event.Data.(map[string]any); ok {
				exchangeEvents = append(exchangeEvents, data)
			}
		}
		if rec.Event.Type == proxy.TypeLLMRequest {
			if data, ok := rec.Event.Data.(map[string]any); ok {
				requestEvents = append(requestEvents, data)
			}
		}
	}

	if len(exchangeEvents) != 2 {
		t.Fatalf("expected 2 exchange.complete events, got %d", len(exchangeEvents))
	}
	if len(requestEvents) != 2 {
		t.Fatalf("expected 2 llm.request events, got %d", len(requestEvents))
	}

	// Verify first exchange event
	ex1 := exchangeEvents[0]
	if ex1["session_id"] == "" {
		t.Fatal("exchange 1 session_id is empty")
	}
	if ex1["exchange_id"] != "0001" {
		t.Fatalf("exchange 1 exchange_id = %v, want 0001", ex1["exchange_id"])
	}
	if ex1["request_event_id"] == "" {
		t.Fatal("exchange 1 request_event_id is empty")
	}
	if got, _ := ex1["actor_id"].(string); got != "test-actor-id" {
		t.Fatalf("exchange 1 actor_id = %v, want test-actor-id", ex1["actor_id"])
	}
	if ex1["completed_at"] == "" {
		t.Fatal("exchange 1 completed_at is empty")
	}
	if ex1["input_tokens"] != float64(10) && ex1["input_tokens"] != int(10) {
		t.Fatalf("exchange 1 input_tokens = %v, want 10", ex1["input_tokens"])
	}
	if ex1["output_tokens"] != float64(20) && ex1["output_tokens"] != int(20) {
		t.Fatalf("exchange 1 output_tokens = %v, want 20", ex1["output_tokens"])
	}

	// Verify second exchange event
	ex2 := exchangeEvents[1]
	if ex2["session_id"] == "" {
		t.Fatal("exchange 2 session_id is empty")
	}
	if ex2["exchange_id"] != "0002" {
		t.Fatalf("exchange 2 exchange_id = %v, want 0002", ex2["exchange_id"])
	}
	if ex2["request_event_id"] == "" {
		t.Fatal("exchange 2 request_event_id is empty")
	}

	// Verify request_event_id matches
	// This requires getting the actual hash of the request events.
	// For now, we just check if they are not empty.
	// A more robust test would re-calculate the hash and compare.
	// However, the current implementation of EventHash is internal and not easily accessible here.
	// The primary goal is to ensure the field is populated.
	if ex1["request_event_id"] == ex2["request_event_id"] {
		t.Fatal("request_event_id for different exchanges should be different")
	}

	// Bodies in this test carry no tool calls, so the count is a real 0.
	if got, _ := ex1["tool_calls_count"].(float64); got != 0 {
		t.Fatalf("exchange 1 tool_calls_count = %v, want 0", ex1["tool_calls_count"])
	}
}

func TestCountToolCallsFromResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		want int
	}{
		{
			name: "anthropic two tool_use blocks",
			body: `{"content":[{"type":"text","text":"hi"},{"type":"tool_use","name":"search"},{"type":"tool_use","name":"fetch"}]}`,
			want: 2,
		},
		{
			name: "openai chat completions tool_calls",
			body: `{"choices":[{"message":{"tool_calls":[{"id":"1"},{"id":"2"},{"id":"3"}]}}]}`,
			want: 3,
		},
		{
			name: "openai responses function_call items",
			body: `{"output":[{"type":"message"},{"type":"function_call","name":"a"},{"type":"tool_call","name":"b"}]}`,
			want: 2,
		},
		{
			name: "no tool calls",
			body: `{"content":[{"type":"text","text":"hi"}],"usage":{"output_tokens":5}}`,
			want: 0,
		},
		{
			name: "unparseable body",
			body: `not json`,
			want: 0,
		},
		{
			name: "empty body",
			body: ``,
			want: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := proxy.CountToolCallsFromResponse([]byte(tc.body)); got != tc.want {
				t.Fatalf("CountToolCallsFromResponse(%s) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}

func TestSessionCloseRecordActorID(t *testing.T) {
	t.Parallel()

	// Create a dummy recorder to capture the session close event
	var capturedSessionClose map[string]any
	baseRecorder := proxy.NewBundleRecorder("testdata/bundle.atb", nil)
	recorder := &testRecorder{
		BundleRecorder: baseRecorder,
		AppendSessionCloseFunc: func(sess *proxy.Session) error {
			t.Logf("AppendSessionCloseFunc called for session %s", sess.ID)
			capturedSessionClose = proxy.SessionCloseRecord(sess)
			return nil
		},
	}

	// Create a SessionManager with the test recorder's callback
	sm := proxy.NewSessionManager(recorder.AppendSessionClose)

	// Resolve a session and set its actor ID
	resolvedSession := sm.Resolve("manual-thread-key")
	resolvedSession.ActorID = "test-actor-id"
	resolvedSession.ID = "manual-sess-actor-test" // Override the generated ID for assertion

	// Simulate some activity to ensure the session is "active"
	sm.NoteExchange(resolvedSession, "gpt-3.5-turbo", 10, 20)

	// Explicitly close all sessions
	sm.CloseAll()

	// Assert the captured session close event
	if capturedSessionClose == nil {
		t.Fatal("expected session close event to be captured")
	}
	if got, ok := capturedSessionClose["actor_id"].(string); !ok || got != "test-actor-id" {
		t.Fatalf("expected actor_id 'test-actor-id', got %v", capturedSessionClose["actor_id"])
	}
	if got, ok := capturedSessionClose["session_id"].(string); !ok || got != "manual-sess-actor-test" {
		t.Fatalf("expected session_id 'manual-sess-actor-test', got %v", capturedSessionClose["session_id"])
	}
}
