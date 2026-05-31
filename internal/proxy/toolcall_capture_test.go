// SPDX-License-Identifier: MIT
package proxy_test

import (
	"bytes"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/proxy"
)

// TestCaptureRecordsToolCallAndActionError drives a request carrying a failed
// Anthropic tool_result and a response requesting a tool call through the
// capture path, and asserts the proxy writes the matching accountability
// events: ai.action.error for the failed tool result and atb.tool.call for the
// requested tool. This is the proxy acting as a real writer for the forensic
// accountability events, not just LLM request/response envelopes.
func TestCaptureRecordsToolCallAndActionError(t *testing.T) {
	host := "api.anthropic.com"
	reqBody := []byte(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"toolu_42","is_error":true,"content":"database unreachable"}` +
		`]}]}`)
	respBody := []byte(`{"model":"claude-3-5-sonnet","content":[` +
		`{"type":"text","text":"retrying"},` +
		`{"type":"tool_use","name":"retry_query","input":{"attempt":2}}` +
		`]}`)

	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "capture.atb")
	recorder := proxy.NewBundleRecorder(bundlePath, nil)

	req, err := http.NewRequest(http.MethodPost, "http://"+host+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody}

	cfg := proxy.ProxyConfig{
		ListenAddr:  "127.0.0.1:0",
		BundlePath:  bundlePath,
		TargetHosts: []string{host},
	}
	if _, err := proxy.RoundTripFixture(host, req, reqBody, resp, respBody, recorder, cfg); err != nil {
		t.Fatalf("RoundTripFixture: %v", err)
	}

	loaded, err := bundle.LoadVerified(bundlePath)
	if err != nil {
		t.Fatalf("LoadVerified: %v", err)
	}

	var sawToolCall, sawActionError bool
	for _, rec := range loaded.Records {
		switch rec.Event.Type {
		case "atb.tool.call":
			sawToolCall = true
			data, _ := rec.Event.Data.(map[string]any)
			if data["tool_name"] != "retry_query" {
				t.Errorf("atb.tool.call tool_name = %v, want retry_query", data["tool_name"])
			}
			if _, ok := data["tool_input_digest"]; !ok {
				t.Error("atb.tool.call missing tool_input_digest")
			}
			if _, raw := data["input"]; raw {
				t.Error("atb.tool.call must not store raw tool input")
			}
		case "ai.action.error":
			sawActionError = true
			data, _ := rec.Event.Data.(map[string]any)
			if data["action_id"] != "toolu_42" {
				t.Errorf("ai.action.error action_id = %v, want toolu_42", data["action_id"])
			}
			if data["error_class"] != "failed" {
				t.Errorf("ai.action.error error_class = %v, want failed", data["error_class"])
			}
			if sid, _ := data["session_id"].(string); sid == "" {
				t.Error("ai.action.error must carry session_id for correlation")
			}
		}
	}
	if !sawToolCall {
		t.Error("expected an atb.tool.call event from the captured response")
	}
	if !sawActionError {
		t.Error("expected an ai.action.error event from the failed tool_result")
	}
}
