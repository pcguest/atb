package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestServeInitialize(t *testing.T) {
	t.Parallel()

	responses := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"claude","version":"1.0"}}}`+"\n")
	if len(responses) != 1 {
		t.Fatalf("unexpected response count: got %d want 1", len(responses))
	}

	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(responses[0].Result, &result); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}

	if result.ProtocolVersion != protocolVersion {
		t.Fatalf("unexpected protocolVersion: got %q want %q", result.ProtocolVersion, protocolVersion)
	}
	if result.ServerInfo.Name != "atb" {
		t.Fatalf("unexpected serverInfo.name: got %q want %q", result.ServerInfo.Name, "atb")
	}
}

func TestServeToolsList(t *testing.T) {
	t.Parallel()

	responses := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`+"\n")
	if len(responses) != 1 {
		t.Fatalf("unexpected response count: got %d want 1", len(responses))
	}

	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(responses[0].Result, &result); err != nil {
		t.Fatalf("unmarshal tools/list result: %v", err)
	}

	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}

	for _, name := range []string{"verify", "bundle", "status"} {
		if !names[name] {
			t.Fatalf("tool %q missing from tools/list response", name)
		}
	}
}

func TestServeToolsCallStatus(t *testing.T) {
	t.Parallel()

	responses := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"status","arguments":{}}}`+"\n")
	if len(responses) != 1 {
		t.Fatalf("unexpected response count: got %d want 1", len(responses))
	}

	var result toolResponse
	if err := json.Unmarshal(responses[0].Result, &result); err != nil {
		t.Fatalf("unmarshal tools/call status result: %v", err)
	}

	if result.IsError {
		t.Fatalf("unexpected isError=true for status tool")
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content count: got %d want 1", len(result.Content))
	}
	if !strings.Contains(result.Content[0].Text, "ATB v") {
		t.Fatalf("unexpected status text: %q", result.Content[0].Text)
	}
}

func TestServeUnknownMethod(t *testing.T) {
	t.Parallel()

	responses := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"nope","params":{}}`+"\n")
	if len(responses) != 1 {
		t.Fatalf("unexpected response count: got %d want 1", len(responses))
	}

	if responses[0].Error == nil {
		t.Fatalf("expected JSON-RPC error response")
	}
	if responses[0].Error.Code != -32601 {
		t.Fatalf("unexpected error code: got %d want %d", responses[0].Error.Code, -32601)
	}
}

func TestServeUnknownTool(t *testing.T) {
	t.Parallel()

	responses := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nonexistent","arguments":{}}}`+"\n")
	if len(responses) != 1 {
		t.Fatalf("unexpected response count: got %d want 1", len(responses))
	}

	var result toolResponse
	if err := json.Unmarshal(responses[0].Result, &result); err != nil {
		t.Fatalf("unmarshal tools/call unknown tool result: %v", err)
	}

	if !result.IsError {
		t.Fatalf("expected isError=true for unknown tool")
	}
}

// TODO: add verify and bundle tool exec-path tests once Session 2 replaces shell-out calls.
func runServer(t *testing.T, input string) []rpcResponse {
	t.Helper()

	var output strings.Builder
	srv := New("1.0.0.1", strings.NewReader(input), &output)
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}

	responses := make([]rpcResponse, 0, len(lines))
	for _, line := range lines {
		var response rpcResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("unmarshal response line %q: %v", line, err)
		}
		responses = append(responses, response)
	}

	return responses
}
