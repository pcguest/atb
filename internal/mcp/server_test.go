// SPDX-License-Identifier: MIT
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/verify"
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

	for _, name := range []string{"verify", "atb_init", "status", "rag_index_record", "rag_retrieval_record"} {
		if !names[name] {
			t.Fatalf("tool %q missing from tools/list response", name)
		}
	}
}

func TestServeToolsCallStatusNoBundle(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

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
	var status map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &status); err != nil {
		t.Fatalf("unmarshal status content: %v", err)
	}
	if bundlePresent, ok := status["bundle_present"].(bool); !ok || bundlePresent {
		t.Fatalf("unexpected bundle_present value: %#v", status["bundle_present"])
	}
	if _, exists := status["chain_length"]; exists {
		t.Fatalf("chain_length should be absent when no bundle is present: %#v", status)
	}
	if !strings.Contains(result.Content[0].Text, `"bundle_present": false`) {
		t.Fatalf("unexpected status text: %q", result.Content[0].Text)
	}
}

func TestServeToolsCallStatusWithBundle(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	if err := b.Save(bundle.DefaultPath()); err != nil {
		t.Fatalf("bundle.Save: %v", err)
	}

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

	var status map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &status); err != nil {
		t.Fatalf("unmarshal status content: %v", err)
	}
	if bundlePresent, ok := status["bundle_present"].(bool); !ok || !bundlePresent {
		t.Fatalf("unexpected bundle_present value: %#v", status["bundle_present"])
	}
	if chainLength, ok := status["chain_length"].(float64); !ok || chainLength < 1 {
		t.Fatalf("unexpected chain_length value: %#v", status["chain_length"])
	}
	if headHash, ok := status["head_hash"].(string); !ok || headHash == "" {
		t.Fatalf("unexpected head_hash value: %#v", status["head_hash"])
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

func TestServeRAGIndexRecord(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

	callTool(t, testToolHandlers(), "atb_init", map[string]any{})
	result := callTool(t, testToolHandlers(), "rag_index_record", map[string]any{
		"index_id":   "idx-test-001",
		"source_uri": "/docs/annual-report.pdf",
		"page_count": 147,
		"node_count": 23,
		"model_id":   "gpt-4o-2024-11-20",
		"index_hash": "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
	})

	if result.IsError {
		t.Fatalf("unexpected isError=true for rag_index_record")
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content count: got %d want 1", len(result.Content))
	}
	if !strings.Contains(result.Content[0].Text, `"status": "ok"`) {
		t.Fatalf("unexpected response text: %q", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, `"event_type": "atb.event.rag_index"`) {
		t.Fatalf("unexpected response text: %q", result.Content[0].Text)
	}

	content := decodeToolContentMap(t, result)
	if indexID, ok := content["index_id"].(string); !ok || indexID != "idx-test-001" {
		t.Fatalf("unexpected index_id: %#v", content["index_id"])
	}
	if sequence, ok := content["sequence"].(float64); !ok || sequence < 1 {
		t.Fatalf("unexpected sequence: %#v", content["sequence"])
	}
	if hash, ok := content["hash"].(string); !ok || hash == "" {
		t.Fatalf("unexpected hash: %#v", content["hash"])
	}
}

func TestServeRAGRetrievalRecord(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

	callTool(t, testToolHandlers(), "atb_init", map[string]any{})
	result := callTool(t, testToolHandlers(), "rag_retrieval_record", map[string]any{
		"query":        "What were net interest margins in Q3?",
		"retrieval_id": "ret-test-001",
		"index_id":     "idx-test-001",
		"node_id":      "0007",
		"node_title":   "Financial Stability",
		"source_uri":   "/docs/annual-report.pdf",
		"page_start":   22,
		"page_end":     28,
		"node_summary": "The Federal Reserve monitored stability metrics.",
		"model_id":     "gpt-4o-2024-11-20",
		"latency_ms":   340,
	})

	if result.IsError {
		t.Fatalf("unexpected isError=true for rag_retrieval_record")
	}
	if !strings.Contains(result.Content[0].Text, `"event_type": "atb.event.rag_retrieval"`) {
		t.Fatalf("unexpected response text: %q", result.Content[0].Text)
	}

	content := decodeToolContentMap(t, result)
	if retrievalID, ok := content["retrieval_id"].(string); !ok || retrievalID != "ret-test-001" {
		t.Fatalf("unexpected retrieval_id: %#v", content["retrieval_id"])
	}
	if indexID, ok := content["index_id"].(string); !ok || indexID != "idx-test-001" {
		t.Fatalf("unexpected index_id: %#v", content["index_id"])
	}
	if sequence, ok := content["sequence"].(float64); !ok || sequence < 1 {
		t.Fatalf("unexpected sequence: %#v", content["sequence"])
	}
}

func TestServeRAGIndexRecordMissingRequired(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

	callTool(t, testToolHandlers(), "atb_init", map[string]any{})
	result := callTool(t, testToolHandlers(), "rag_index_record", map[string]any{
		"source_uri": "/docs/annual-report.pdf",
		"page_count": 147,
		"node_count": 23,
		"model_id":   "gpt-4o-2024-11-20",
		"index_hash": "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
	})

	if !result.IsError {
		t.Fatalf("expected isError=true for missing required field")
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content count: got %d want 1", len(result.Content))
	}
	if !strings.Contains(result.Content[0].Text, "index_id") && !strings.Contains(result.Content[0].Text, "required") && !strings.Contains(result.Content[0].Text, "missing") {
		t.Fatalf("unexpected error text: %q", result.Content[0].Text)
	}
}

func TestServeRAGChainIntegrity(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

	callTool(t, testToolHandlers(), "atb_init", map[string]any{})

	indexResult := callTool(t, testToolHandlers(), "rag_index_record", map[string]any{
		"index_id":   "chain-test-idx",
		"source_uri": "/docs/annual-report.pdf",
		"page_count": 147,
		"node_count": 23,
		"model_id":   "gpt-4o-2024-11-20",
		"index_hash": "deadbeef01234567deadbeef01234567deadbeef01234567deadbeef01234567",
	})
	if indexResult.IsError {
		t.Fatalf("unexpected isError=true for rag_index_record")
	}
	indexContent := decodeToolContentMap(t, indexResult)
	if sequence, ok := indexContent["sequence"].(float64); !ok || sequence != 1 {
		t.Fatalf("unexpected rag_index_record sequence: %#v", indexContent["sequence"])
	}

	retrievalResult := callTool(t, testToolHandlers(), "rag_retrieval_record", map[string]any{
		"query":        "What were net interest margins?",
		"retrieval_id": "chain-test-ret",
		"index_id":     "chain-test-idx",
		"node_id":      "0007",
		"node_title":   "Financial Stability",
		"source_uri":   "/docs/annual-report.pdf",
		"page_start":   22,
		"page_end":     28,
		"model_id":     "gpt-4o-2024-11-20",
		"latency_ms":   312,
	})
	if retrievalResult.IsError {
		t.Fatalf("unexpected isError=true for rag_retrieval_record")
	}
	retrievalContent := decodeToolContentMap(t, retrievalResult)
	if sequence, ok := retrievalContent["sequence"].(float64); !ok || sequence != 2 {
		t.Fatalf("unexpected rag_retrieval_record sequence: %#v", retrievalContent["sequence"])
	}

	statusResult := callTool(t, testToolHandlers(), "status", map[string]any{})
	if statusResult.IsError {
		t.Fatalf("unexpected isError=true for status")
	}
	statusContent := decodeToolContentMap(t, statusResult)
	if bundlePresent, ok := statusContent["bundle_present"].(bool); !ok || !bundlePresent {
		t.Fatalf("unexpected bundle_present value: %#v", statusContent["bundle_present"])
	}
	if chainLength, ok := statusContent["chain_length"].(float64); !ok || chainLength != 3 {
		t.Fatalf("unexpected chain_length value: %#v", statusContent["chain_length"])
	}
	if headHash, ok := statusContent["head_hash"].(string); !ok || headHash == "" {
		t.Fatalf("unexpected head_hash value: %#v", statusContent["head_hash"])
	}
}

func TestServeMCPVerifyIntegration(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

	handlers := ToolHandlers{
		Init:   testInitHandler,
		Append: testAppendHandler,
		Verify: testVerifyHandler,
	}

	// Initialise the bundle via MCP.
	initResult := callTool(t, handlers, "atb_init", map[string]any{})
	if initResult.IsError {
		t.Fatalf("unexpected isError=true for atb_init: %s", initResult.Content[0].Text)
	}

	// Append one event directly via the bundle package.
	b, err := bundle.Load(bundle.DefaultPath())
	if err != nil {
		t.Fatalf("bundle.Load: %v", err)
	}
	if err := b.AppendWithOptions("ai.request.received", map[string]any{
		"request_id":    "req-mcp-smoke-001",
		"actor_id_hash": "sha256-test-actor",
		"purpose_tag":   "audit",
	}, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("b.AppendWithOptions: %v", err)
	}
	if err := b.Save(bundle.DefaultPath()); err != nil {
		t.Fatalf("b.Save: %v", err)
	}

	// Verify via MCP and assert chain is valid.
	verifyResult := callTool(t, handlers, "verify", map[string]any{})
	if verifyResult.IsError {
		t.Fatalf("unexpected isError=true for verify: %s", verifyResult.Content[0].Text)
	}
	if len(verifyResult.Content) != 1 {
		t.Fatalf("unexpected content count: got %d want 1", len(verifyResult.Content))
	}

	var content map[string]any
	if err := json.Unmarshal([]byte(verifyResult.Content[0].Text), &content); err != nil {
		t.Fatalf("unmarshal verify content: %v", err)
	}
	if status, ok := content["status"].(string); !ok || status != "ok" {
		t.Fatalf("unexpected status: %#v (want \"ok\")", content["status"])
	}
	exitCode, ok := content["exit_code"].(float64)
	if !ok || exitCode != 0 {
		t.Fatalf("unexpected exit_code: %#v (want 0, implies chain_valid=true)", content["exit_code"])
	}
	report, ok := content["report"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected report type: %T", content["report"])
	}
	failures, ok := report["critical_failures"].([]any)
	if !ok {
		t.Fatalf("unexpected critical_failures type: %T", report["critical_failures"])
	}
	if len(failures) != 0 {
		t.Fatalf("unexpected critical_failures (want empty): %v", failures)
	}
}

func testVerifyHandler(_ context.Context, input VerifyInput, stdout, stderr io.Writer) int {
	path := bundle.DefaultPath()
	if input.Path != "" {
		path = input.Path
	}

	b, err := bundle.Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "bundle.Load: %v\n", err)
		return 3
	}

	selection := verify.EvaluateConfig{
		BundlePath:    path,
		Records:       b.Records,
		AllApplicable: strings.TrimSpace(input.Profile) == "",
	}
	if strings.TrimSpace(input.Profile) != "" {
		profile, err := verify.ResolveProfile(input.Profile)
		if err != nil {
			fmt.Fprintf(stderr, "resolve profile: %v\n", err)
			return 3
		}
		selection.Profiles = []verify.Profile{profile}
	}

	report, err := verify.EvaluateBundle(selection)
	if err != nil {
		fmt.Fprintf(stderr, "EvaluateBundle: %v\n", err)
		return 3
	}
	verifierReport := verify.ReportFromVerify(*report)

	data, err := json.MarshalIndent(verifierReport, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "json.Marshal: %v\n", err)
		return 3
	}
	data = append(data, '\n')
	if _, err := stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "write: %v\n", err)
		return 3
	}

	if !report.Integrity.ChainValid {
		return 2
	}
	return 0
}

// TestServeRAGToolsRoundTrip exercises the full rag_index_record →
// rag_retrieval_record dispatch path through the JSON-RPC server and asserts
// that each response carries a non-empty hash and a sequence number >= 1.
// This test covers the complete stdio round-trip, not just the tool handlers.
func TestServeRAGToolsRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	restore := chdirTempDir(t, tempDir)
	defer restore()

	handlers := testToolHandlers()

	// Initialise a bundle first.
	callTool(t, handlers, "atb_init", map[string]any{})

	// Send rag_index_record and assert seq + hash.
	indexResult := callTool(t, handlers, "rag_index_record", map[string]any{
		"index_id":   "idx-rt-001",
		"source_uri": "/docs/roundtrip.pdf",
		"page_count": 10,
		"node_count": 4,
		"model_id":   "gpt-4o-mini",
		"index_hash": "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd",
	})
	if indexResult.IsError {
		t.Fatalf("rag_index_record: unexpected error: %v", indexResult.Content)
	}
	indexContent := decodeToolContentMap(t, indexResult)
	if seq, ok := indexContent["sequence"].(float64); !ok || seq < 1 {
		t.Fatalf("rag_index_record: expected sequence >= 1, got %#v", indexContent["sequence"])
	}
	if h, ok := indexContent["hash"].(string); !ok || h == "" {
		t.Fatalf("rag_index_record: expected non-empty hash, got %#v", indexContent["hash"])
	}

	// Send rag_retrieval_record and assert seq + hash.
	retrievalResult := callTool(t, handlers, "rag_retrieval_record", map[string]any{
		"query":        "What is the summary?",
		"retrieval_id": "ret-rt-001",
		"index_id":     "idx-rt-001",
		"node_id":      "0001",
		"node_title":   "Introduction",
		"source_uri":   "/docs/roundtrip.pdf",
		"page_start":   1,
		"page_end":     2,
		"node_summary": "Summary text.",
		"model_id":     "gpt-4o-mini",
		"latency_ms":   50,
	})
	if retrievalResult.IsError {
		t.Fatalf("rag_retrieval_record: unexpected error: %v", retrievalResult.Content)
	}
	retrievalContent := decodeToolContentMap(t, retrievalResult)
	if seq, ok := retrievalContent["sequence"].(float64); !ok || seq < 1 {
		t.Fatalf("rag_retrieval_record: expected sequence >= 1, got %#v", retrievalContent["sequence"])
	}
	if h, ok := retrievalContent["hash"].(string); !ok || h == "" {
		t.Fatalf("rag_retrieval_record: expected non-empty hash, got %#v", retrievalContent["hash"])
	}
}

func runServer(t *testing.T, input string) []rpcResponse {
	return runServerWithHandlers(t, ToolHandlers{}, input)
}

func runServerWithHandlers(t *testing.T, handlers ToolHandlers, input string) []rpcResponse {
	t.Helper()

	var output strings.Builder
	srv := NewWithHandlers("1.0.0.1", strings.NewReader(input), &output, handlers)
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

func callTool(t *testing.T, handlers ToolHandlers, name string, arguments map[string]any) toolResponse {
	t.Helper()

	payload, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("json.Marshal arguments: %v", err)
	}
	request := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"%s","arguments":%s}}`+"\n", name, payload)
	responses := runServerWithHandlers(t, handlers, request)
	if len(responses) != 1 {
		t.Fatalf("unexpected response count: got %d want 1", len(responses))
	}

	var result toolResponse
	if err := json.Unmarshal(responses[0].Result, &result); err != nil {
		t.Fatalf("unmarshal tools/call result: %v", err)
	}
	return result
}

func decodeToolContentMap(t *testing.T, result toolResponse) map[string]any {
	t.Helper()

	if len(result.Content) != 1 {
		t.Fatalf("unexpected content count: got %d want 1", len(result.Content))
	}

	var content map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &content); err != nil {
		t.Fatalf("unmarshal tool content: %v", err)
	}
	return content
}

func testToolHandlers() ToolHandlers {
	return ToolHandlers{
		Init:   testInitHandler,
		Append: testAppendHandler,
	}
}

func testInitHandler(_ context.Context, stdout, stderr io.Writer) int {
	b, err := bundle.New()
	if err != nil {
		fmt.Fprintf(stderr, "bundle.New: %v", err)
		return 3
	}
	if err := b.Save(bundle.DefaultPath()); err != nil {
		fmt.Fprintf(stderr, "bundle.Save: %v", err)
		return 3
	}
	return writeJSONResult(stdout, map[string]any{
		"status":  "ok",
		"action":  "init",
		"path":    bundle.DefaultPath(),
		"message": "bundle initialised",
	}, stderr)
}

func testAppendHandler(_ context.Context, eventType string, data map[string]any) (bundle.Record, error) {
	path := bundle.DefaultPath()
	b, err := bundle.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			b, err = bundle.New()
			if err != nil {
				return bundle.Record{}, err
			}
		} else {
			return bundle.Record{}, err
		}
	}

	if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return bundle.Record{}, err
	}
	last := b.Records[len(b.Records)-1]
	if err := b.Save(path); err != nil {
		return bundle.Record{}, err
	}
	return last, nil
}

func writeJSONResult(stdout io.Writer, payload map[string]any, stderr io.Writer) int {
	if err := json.NewEncoder(stdout).Encode(payload); err != nil {
		fmt.Fprintf(stderr, "encode json output: %v", err)
		return 3
	}
	return 0
}

func chdirTempDir(t *testing.T, dir string) func() {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q): %v", dir, err)
	}

	return func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore os.Chdir(%q): %v", cwd, err)
		}
	}
}
