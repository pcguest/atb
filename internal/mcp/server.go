// SPDX-License-Identifier: MIT
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

const protocolVersion = "2024-11-05"

type VerifyInput struct {
	Path       string `json:"path,omitempty"`
	Profile    string `json:"profile,omitempty"`
	Trace      bool   `json:"trace,omitempty"`
	WithAnchor bool   `json:"with_anchor,omitempty"`
	Quiet      bool   `json:"quiet,omitempty"`
}

type VerifyHandler func(context.Context, VerifyInput, io.Writer, io.Writer) int

type InitHandler func(context.Context, io.Writer, io.Writer) int

// MCP RAG tool notes:
// 1. handleToolsList registers tools inline via a []toolDefinition slice literal.
// 2. cmd/atb appendToDefaultBundle is called as appendToDefaultBundle(eventType string, data interface{}, dryRun bool, opts ...bundle.AppendOptions) (bundle.Record, error), so the CLI-side append handler returns the appended bundle.Record for sequence/hash reporting here.
// 3. The existing status tool returns a structured JSON object such as {"version":"...","bundle_present":false} and adds chain_length/head_hash when a bundle exists before wrapping it with newStructuredToolResponse.
// 4. Input validation in this server is manual after json.Unmarshal; inputSchema is descriptive only, so required-field and type checks for the RAG tools must stay in code.
type AppendHandler func(context.Context, string, map[string]any) (bundle.Record, error)

type ToolHandlers struct {
	Verify VerifyHandler
	Init   InitHandler
	Append AppendHandler
}

type Server struct {
	version string
	in      *bufio.Reader
	out     *bufio.Writer
	mu      sync.Mutex

	ctx      context.Context
	handlers ToolHandlers
}

type request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type initParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
	ClientInfo      clientInfo     `json:"clientInfo,omitempty"`
}

type clientInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolResponse struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type readResult struct {
	line []byte
	err  error
}

func New(version string, in io.Reader, out io.Writer) *Server {
	return NewWithHandlers(version, in, out, ToolHandlers{})
}

func NewWithHandlers(version string, in io.Reader, out io.Writer, handlers ToolHandlers) *Server {
	return &Server{
		version:  version,
		in:       bufio.NewReader(in),
		out:      bufio.NewWriter(out),
		handlers: handlers,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.ctx = ctx

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	results := make(chan readResult, 1)

	go func() {
		for {
			line, err := s.in.ReadBytes('\n')
			results <- readResult{line: line, err: err}
			if err != nil {
				close(results)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case res, ok := <-results:
			if !ok {
				return nil
			}

			raw := bytes.TrimSpace(res.line)
			if len(raw) > 0 {
				if err := s.handleMessage(raw); err != nil {
					logger.Debug("mcp handle message", "error", err)
				}
			}

			switch {
			case res.err == nil:
				continue
			case res.err == io.EOF:
				return nil
			default:
				logger.Debug("mcp read message", "error", res.err)
				return res.err
			}
		}
	}
}

func (s *Server) handleMessage(raw []byte) error {
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		if respondErr := s.respondError(nil, -32700, "parse error"); respondErr != nil {
			return fmt.Errorf("respond parse error: %w", respondErr)
		}
		return nil
	}

	if req.JSONRPC != "2.0" {
		return s.respondError(req.ID, -32600, "invalid request")
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID, req.Params)
	case "notifications/initialized":
		return nil
	case "tools/list":
		return s.handleToolsList(req.ID)
	case "tools/call":
		return s.handleToolsCall(req.ID, req.Params)
	default:
		return s.respondError(req.ID, -32601, "method not found")
	}
}

func (s *Server) handleInitialize(id *json.RawMessage, raw json.RawMessage) error {
	var params initParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return s.respondError(id, -32602, "invalid params")
		}
	}

	version := protocolVersion
	if params.ProtocolVersion == protocolVersion {
		version = params.ProtocolVersion
	}

	result := map[string]any{
		"protocolVersion": version,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "atb",
			"version": s.version,
		},
	}
	return s.respond(id, result)
}

func (s *Server) handleToolsList(id *json.RawMessage) error {
	result := map[string]any{
		"tools": []toolDefinition{
			{
				Name:        "verify",
				Description: "Verify an ATB bundle's integrity and trust chain",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the .atb bundle file",
						},
						"profile": map[string]any{
							"type":        "string",
							"description": "Profile ID or path to a YAML profile file",
						},
						"trace": map[string]any{
							"type":        "boolean",
							"description": "Emit trace diagnostics during verification",
						},
						"with_anchor": map[string]any{
							"type":        "boolean",
							"description": "Verify RFC 3161 timestamp token material when present",
						},
						"quiet": map[string]any{
							"type":        "boolean",
							"description": "Suppress non-JSON terminal chatter",
						},
					},
					"additionalProperties": false,
				},
			},
			{
				Name:        "atb_init",
				Description: "Initialise a new ATB bundle at the current working directory (idempotent)",
				InputSchema: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
			{
				Name:        "status",
				Description: "Return ATB server status, version, and local bundle state",
				InputSchema: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
			{
				Name:        "rag_index_record",
				Description: "Record a PageIndex document tree build and append atb.event.rag_index to the current bundle",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"index_id": map[string]any{
							"type":        "string",
							"description": "Unique identifier for this index build",
						},
						"source_uri": map[string]any{
							"type":        "string",
							"description": "Absolute path or URI of the indexed document",
						},
						"page_count": map[string]any{
							"type":        "integer",
							"description": "Total pages in the document",
						},
						"node_count": map[string]any{
							"type":        "integer",
							"description": "Total nodes in the PageIndex tree (recursive count)",
						},
						"model_id": map[string]any{
							"type":        "string",
							"description": "LLM model used to build the index",
						},
						"index_hash": map[string]any{
							"type":        "string",
							"description": "SHA-256 hex digest of json.dumps(tree, sort_keys=True)",
						},
						"indexed_at": map[string]any{
							"type":        "string",
							"description": "RFC3339 timestamp — defaults to server time if omitted",
						},
					},
					"required": []string{
						"index_id",
						"source_uri",
						"page_count",
						"node_count",
						"model_id",
						"index_hash",
					},
					"additionalProperties": false,
				},
			},
			{
				Name:        "rag_retrieval_record",
				Description: "Record a PageIndex reasoning-based retrieval result and append atb.event.rag_retrieval to the current bundle",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type": "string",
						},
						"retrieval_id": map[string]any{
							"type": "string",
						},
						"index_id": map[string]any{
							"type":        "string",
							"description": "Foreign key — must match a prior rag_index_record.index_id",
						},
						"node_id": map[string]any{
							"type":        "string",
							"description": "PageIndex node_id from the matched tree node",
						},
						"node_title": map[string]any{
							"type":        "string",
							"description": "PageIndex title field from the matched tree node",
						},
						"source_uri": map[string]any{
							"type": "string",
						},
						"page_start": map[string]any{
							"type":        "integer",
							"description": "PageIndex start_index of the matched node",
						},
						"page_end": map[string]any{
							"type":        "integer",
							"description": "PageIndex end_index of the matched node",
						},
						"node_summary": map[string]any{
							"type":        "string",
							"description": "PageIndex summary field — omit if absent",
						},
						"model_id": map[string]any{
							"type": "string",
						},
						"latency_ms": map[string]any{
							"type":        "integer",
							"description": "Wall-clock milliseconds for the retrieval call",
						},
					},
					"required": []string{
						"query",
						"retrieval_id",
						"index_id",
						"node_id",
						"node_title",
						"source_uri",
						"page_start",
						"page_end",
						"model_id",
						"latency_ms",
					},
					"additionalProperties": false,
				},
			},
		},
	}

	return s.respond(id, result)
}

func (s *Server) handleToolsCall(id *json.RawMessage, raw json.RawMessage) error {
	var params callParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return s.respondError(id, -32602, "invalid params")
	}

	var (
		result toolResponse
		err    error
	)
	switch params.Name {
	case "verify":
		result, err = s.toolVerify(params.Arguments)
	case "atb_init":
		result, err = s.toolInit(params.Arguments)
	case "status":
		result, err = s.toolStatus()
	case "rag_index_record":
		result, err = s.toolRAGIndexRecord(params.Arguments)
	case "rag_retrieval_record":
		result, err = s.toolRAGRetrievalRecord(params.Arguments)
	default:
		result = newToolResponse(fmt.Sprintf("unknown tool: %s", params.Name), true)
	}

	if err != nil {
		return s.respondCallError(id, err)
	}

	return s.respond(id, result)
}

func (s *Server) toolStatus() (toolResponse, error) {
	status := map[string]any{
		"version":        s.version,
		"bundle_present": false,
	}

	b, err := bundle.Load(bundle.DefaultPath())
	switch {
	case err == nil:
		status["bundle_present"] = true
		status["chain_length"] = len(b.Records)
		if len(b.Records) > 0 {
			status["head_hash"] = abbreviateHash(b.Records[len(b.Records)-1].Hash)
		}
	case os.IsNotExist(err):
		// No bundle is a normal status condition.
	default:
		status["error"] = err.Error()
	}

	return newStructuredToolResponse(status)
}

func (s *Server) toolVerify(raw json.RawMessage) (toolResponse, error) {
	if s.handlers.Verify == nil {
		return toolResponse{}, &callError{Code: -32002, Message: "system error", Data: map[string]any{
			"details":   "verify handler is not configured",
			"exit_code": 3,
		}}
	}

	var args VerifyInput
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return toolResponse{}, &callError{Code: -32602, Message: "invalid params", Data: map[string]any{
				"details": err.Error(),
			}}
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := s.handlers.Verify(s.commandContext(), args, &stdout, &stderr)
	if exitCode != 0 {
		return toolResponse{}, mcpError(exitCode, stderr.String())
	}

	var report any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		return toolResponse{}, &callError{Code: -32002, Message: "system error", Data: map[string]any{
			"details":   fmt.Sprintf("decode verify json: %v", err),
			"exit_code": 3,
		}}
	}

	return newStructuredToolResponse(map[string]any{
		"status":    "ok",
		"exit_code": exitCode,
		"report":    report,
	})
}

func (s *Server) toolInit(raw json.RawMessage) (toolResponse, error) {
	if s.handlers.Init == nil {
		return toolResponse{}, &callError{Code: -32002, Message: "system error", Data: map[string]any{
			"details":   "init handler is not configured",
			"exit_code": 3,
		}}
	}

	if len(raw) > 0 {
		var args map[string]any
		if err := json.Unmarshal(raw, &args); err != nil {
			return toolResponse{}, &callError{Code: -32602, Message: "invalid params", Data: map[string]any{
				"details": err.Error(),
			}}
		}
		if len(args) > 0 {
			return toolResponse{}, &callError{Code: -32602, Message: "invalid params", Data: map[string]any{
				"details": "atb_init does not accept arguments",
			}}
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := s.handlers.Init(s.commandContext(), &stdout, &stderr)
	if exitCode != 0 {
		return toolResponse{}, mcpError(exitCode, stderr.String())
	}

	var result any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return toolResponse{}, &callError{Code: -32002, Message: "system error", Data: map[string]any{
			"details":   fmt.Sprintf("decode init json: %v", err),
			"exit_code": 3,
		}}
	}

	return newStructuredToolResponse(result)
}

func (s *Server) toolRAGIndexRecord(raw json.RawMessage) (toolResponse, error) {
	if s.handlers.Append == nil {
		return newToolResponse("system error: append handler is not configured", true), nil
	}

	args, err := decodeObjectArguments(raw)
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	if err := rejectUnknownFields(args, map[string]struct{}{
		"index_id":   {},
		"source_uri": {},
		"page_count": {},
		"node_count": {},
		"model_id":   {},
		"index_hash": {},
		"indexed_at": {},
	}); err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}

	indexID, err := requireStringField(args, "index_id")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	sourceURI, err := requireStringField(args, "source_uri")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	pageCount, err := requireIntegerField(args, "page_count")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	nodeCount, err := requireIntegerField(args, "node_count")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	modelID, err := requireStringField(args, "model_id")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	indexHash, err := requireStringField(args, "index_hash")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}

	indexedAt, present, err := optionalStringField(args, "indexed_at")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	if !present || strings.TrimSpace(indexedAt) == "" {
		// Default server-side so the hashed event payload is complete even when clients omit the timestamp.
		indexedAt = time.Now().UTC().Format(time.RFC3339)
	}

	data := map[string]any{
		"index_id":   indexID,
		"source_uri": sourceURI,
		"page_count": pageCount,
		"node_count": nodeCount,
		"model_id":   modelID,
		"index_hash": indexHash,
		"indexed_at": indexedAt,
	}

	record, err := s.handlers.Append(s.commandContext(), event.TypeRAGIndex, data)
	if err != nil {
		return newToolResponse(fmt.Sprintf("system error: %v", err), true), nil
	}

	return newStructuredToolResponse(map[string]any{
		"status":     "ok",
		"event_type": event.TypeRAGIndex,
		"index_id":   indexID,
		"sequence":   record.Event.Sequence,
		"hash":       abbreviateHash(record.Hash),
	})
}

func (s *Server) toolRAGRetrievalRecord(raw json.RawMessage) (toolResponse, error) {
	if s.handlers.Append == nil {
		return newToolResponse("system error: append handler is not configured", true), nil
	}

	args, err := decodeObjectArguments(raw)
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	if err := rejectUnknownFields(args, map[string]struct{}{
		"query":        {},
		"retrieval_id": {},
		"index_id":     {},
		"node_id":      {},
		"node_title":   {},
		"source_uri":   {},
		"page_start":   {},
		"page_end":     {},
		"node_summary": {},
		"model_id":     {},
		"latency_ms":   {},
	}); err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}

	query, err := requireStringField(args, "query")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	retrievalID, err := requireStringField(args, "retrieval_id")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	indexID, err := requireStringField(args, "index_id")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	nodeID, err := requireStringField(args, "node_id")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	nodeTitle, err := requireStringField(args, "node_title")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	sourceURI, err := requireStringField(args, "source_uri")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	pageStart, err := requireIntegerField(args, "page_start")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	pageEnd, err := requireIntegerField(args, "page_end")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	modelID, err := requireStringField(args, "model_id")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}
	latencyMS, err := requireIntegerField(args, "latency_ms")
	if err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	}

	data := map[string]any{
		"query":        query,
		"retrieval_id": retrievalID,
		"index_id":     indexID,
		"node_id":      nodeID,
		"node_title":   nodeTitle,
		"source_uri":   sourceURI,
		"page_start":   pageStart,
		"page_end":     pageEnd,
		"model_id":     modelID,
		"latency_ms":   latencyMS,
	}
	if nodeSummary, present, err := optionalStringField(args, "node_summary"); err != nil {
		return newToolResponse(fmt.Sprintf("invalid params: %v", err), true), nil
	} else if present && strings.TrimSpace(nodeSummary) != "" {
		data["node_summary"] = nodeSummary
	}

	record, err := s.handlers.Append(s.commandContext(), event.TypeRAGRetrieval, data)
	if err != nil {
		return newToolResponse(fmt.Sprintf("system error: %v", err), true), nil
	}

	return newStructuredToolResponse(map[string]any{
		"status":       "ok",
		"event_type":   event.TypeRAGRetrieval,
		"retrieval_id": retrievalID,
		"index_id":     indexID,
		"sequence":     record.Event.Sequence,
		"hash":         abbreviateHash(record.Hash),
	})
}

func (s *Server) respond(id *json.RawMessage, result any) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      rawMessageValue(id),
		"result":  result,
	}
	return s.writeResponse(payload)
}

func (s *Server) respondError(id *json.RawMessage, code int, message string) error {
	return s.respondErrorWithData(id, code, message, nil)
}

func (s *Server) respondErrorWithData(id *json.RawMessage, code int, message string, data any) error {
	errPayload := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		errPayload["data"] = data
	}
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      rawMessageValue(id),
		"error":   errPayload,
	}
	return s.writeResponse(payload)
}

func (s *Server) writeResponse(payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.out.Write(data); err != nil {
		return err
	}
	if err := s.out.WriteByte('\n'); err != nil {
		return err
	}
	return s.out.Flush()
}

func (s *Server) commandContext() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func newToolResponse(text string, isError bool) toolResponse {
	return toolResponse{
		Content: []toolContent{{
			Type: "text",
			Text: text,
		}},
		IsError: isError,
	}
}

func newStructuredToolResponse(data any) (toolResponse, error) {
	text, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return toolResponse{}, err
	}

	return toolResponse{
		Content: []toolContent{{
			Type: "text",
			Text: string(text),
		}},
		StructuredContent: data,
	}, nil
}

type callError struct {
	Code    int
	Message string
	Data    any
}

func (e *callError) Error() string {
	return e.Message
}

func mcpError(exitCode int, stderr string) error {
	data := map[string]any{
		"exit_code": exitCode,
	}
	if details := strings.TrimSpace(stderr); details != "" {
		data["details"] = details
	}

	switch exitCode {
	case 1:
		return &callError{Code: -32602, Message: "invalid params", Data: data}
	case 2:
		return &callError{Code: -32001, Message: "integrity verification failure", Data: data}
	case 3:
		return &callError{Code: -32002, Message: "system error", Data: data}
	default:
		data["details"] = fmt.Sprintf("unexpected exit code %d", exitCode)
		return &callError{Code: -32002, Message: "system error", Data: data}
	}
}

func (s *Server) respondCallError(id *json.RawMessage, err error) error {
	var callErr *callError
	if !errors.As(err, &callErr) {
		return s.respondError(id, -32002, "system error")
	}
	return s.respondErrorWithData(id, callErr.Code, callErr.Message, callErr.Data)
}

func abbreviateHash(hash string) string {
	if len(hash) <= 16 {
		return hash
	}
	return hash[:16] + "..."
}

func decodeObjectArguments(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var args map[string]any
	if err := decoder.Decode(&args); err != nil {
		return nil, err
	}
	if args == nil {
		return map[string]any{}, nil
	}
	return args, nil
}

func rejectUnknownFields(args map[string]any, allowed map[string]struct{}) error {
	for key := range args {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}

func requireStringField(args map[string]any, field string) (string, error) {
	value, ok := args[field]
	if !ok {
		return "", fmt.Errorf("missing required field %q", field)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("field %q must be a string", field)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("field %q must not be empty", field)
	}
	return text, nil
}

func optionalStringField(args map[string]any, field string) (string, bool, error) {
	value, ok := args[field]
	if !ok {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", false, fmt.Errorf("field %q must be a string", field)
	}
	return text, true, nil
}

func requireIntegerField(args map[string]any, field string) (int, error) {
	value, ok := args[field]
	if !ok {
		return 0, fmt.Errorf("missing required field %q", field)
	}

	switch typed := value.(type) {
	case json.Number:
		n, err := typed.Int64()
		if err != nil {
			return 0, fmt.Errorf("field %q must be an integer", field)
		}
		return int(n), nil
	case float64:
		if typed != float64(int(typed)) {
			return 0, fmt.Errorf("field %q must be an integer", field)
		}
		return int(typed), nil
	default:
		return 0, fmt.Errorf("field %q must be an integer", field)
	}
}

func rawMessageValue(id *json.RawMessage) any {
	if id == nil {
		return nil
	}

	var value any
	if err := json.Unmarshal(*id, &value); err != nil {
		return nil
	}
	return value
}
