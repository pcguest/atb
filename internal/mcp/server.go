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

	"github.com/pcguest/atb/internal/bundle"
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

type ToolHandlers struct {
	Verify VerifyHandler
	Init   InitHandler
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
