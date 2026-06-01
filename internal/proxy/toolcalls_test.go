// SPDX-License-Identifier: MIT
package proxy_test

import (
	"testing"

	"github.com/pcguest/atb/internal/proxy"
)

func TestExtractToolCallsOpenAIChat(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"tool_calls":[
		{"id":"call_1","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}},
		{"id":"call_2","function":{"name":"send_email","arguments":"{\"to\":\"a@b.c\"}"}}
	]}}]}`)
	calls := proxy.ExtractToolCalls(body)
	if len(calls) != 2 {
		t.Fatalf("want 2 tool calls, got %d", len(calls))
	}
	if calls[0].Name != "get_weather" || calls[1].Name != "send_email" {
		t.Fatalf("unexpected tool names: %q, %q", calls[0].Name, calls[1].Name)
	}
	if calls[0].InputDigest() == "" {
		t.Error("expected non-empty input digest for tool args")
	}
}

func TestExtractToolCallsAnthropic(t *testing.T) {
	body := []byte(`{"content":[
		{"type":"text","text":"let me check"},
		{"type":"tool_use","name":"lookup_account","input":{"id":"cust-1"}}
	]}`)
	calls := proxy.ExtractToolCalls(body)
	if len(calls) != 1 {
		t.Fatalf("want 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "lookup_account" {
		t.Fatalf("tool name = %q", calls[0].Name)
	}
}

func TestExtractToolCallsOpenAIResponses(t *testing.T) {
	body := []byte(`{"output":[{"type":"function_call","name":"run_query","arguments":"{\"q\":\"x\"}"}]}`)
	calls := proxy.ExtractToolCalls(body)
	if len(calls) != 1 || calls[0].Name != "run_query" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestExtractToolCallsOpenAIStreaming(t *testing.T) {
	// Tool name arrives in the first delta; arguments stream across chunks.
	body := []byte(
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"get_weather","arguments":"{\"ci"}}]}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ty\":\"Paris\"}"}}]}}]}` + "\n\n" +
			`data: [DONE]` + "\n")
	calls := proxy.ExtractToolCalls(body)
	if len(calls) != 1 || calls[0].Name != "get_weather" {
		t.Fatalf("unexpected streamed calls: %+v", calls)
	}
	if string(calls[0].Arguments) != `{"city":"Paris"}` {
		t.Errorf("reassembled arguments = %q", string(calls[0].Arguments))
	}
}

func TestExtractToolCallsAnthropicStreaming(t *testing.T) {
	body := []byte(
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","name":"lookup_account","input":{}}}` + "\n\n" +
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"id\":\"cust-1\"}"}}` + "\n\n")
	calls := proxy.ExtractToolCalls(body)
	if len(calls) != 1 || calls[0].Name != "lookup_account" {
		t.Fatalf("unexpected streamed calls: %+v", calls)
	}
	if string(calls[0].Arguments) != `{"id":"cust-1"}` {
		t.Errorf("reassembled arguments = %q", string(calls[0].Arguments))
	}
}

func TestExtractToolCallsNoneForPlainResponse(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"hello"}}]}`)
	if got := proxy.ExtractToolCalls(body); len(got) != 0 {
		t.Fatalf("want no tool calls, got %d", len(got))
	}
	if got := proxy.ExtractToolCalls([]byte("not json")); got != nil {
		t.Fatalf("want nil for invalid body, got %+v", got)
	}
}

func TestExtractToolResultErrorsAnthropic(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[
		{"type":"tool_result","tool_use_id":"toolu_1","is_error":true,"content":"connection refused"},
		{"type":"tool_result","tool_use_id":"toolu_2","is_error":false,"content":"ok"}
	]}]}`)
	errs := proxy.ExtractToolResultErrors(body)
	if len(errs) != 1 {
		t.Fatalf("want 1 errored tool result, got %d", len(errs))
	}
	if errs[0].ToolUseID != "toolu_1" {
		t.Fatalf("tool_use_id = %q", errs[0].ToolUseID)
	}
	if errs[0].DetailDigest() == "" {
		t.Error("expected non-empty error detail digest")
	}
}

func TestExtractToolResultErrorsSkipsMissingID(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[
		{"type":"tool_result","is_error":true,"content":"boom"}
	]}]}`)
	if got := proxy.ExtractToolResultErrors(body); len(got) != 0 {
		t.Fatalf("want no errors without tool_use_id, got %d", len(got))
	}
}
