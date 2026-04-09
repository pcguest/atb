package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

const protocolVersion = "2024-11-05"

type Server struct {
	version string
	in      *bufio.Reader
	out     *bufio.Writer
	mu      sync.Mutex

	ctx context.Context
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

type verifyArgs struct {
	Path   string `json:"path"`
	Strict bool   `json:"strict"`
}

type bundleArgs struct {
	Source string `json:"source"`
	Output string `json:"output"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolResponse struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError"`
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
	return &Server{
		version: version,
		in:      bufio.NewReader(in),
		out:     bufio.NewWriter(out),
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
						"strict": map[string]any{
							"type":        "boolean",
							"description": "Fail on warnings (default false)",
						},
					},
					"required": []string{"path"},
				},
			},
			{
				Name:        "bundle",
				Description: "Create an ATB bundle from a directory or file",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"source": map[string]any{
							"type":        "string",
							"description": "Path to the source directory or file",
						},
						"output": map[string]any{
							"type":        "string",
							"description": "Output path for the .atb bundle",
						},
					},
					"required": []string{"source", "output"},
				},
			},
			{
				Name:        "status",
				Description: "Return ATB server status and version",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
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

	var result toolResponse
	switch params.Name {
	case "verify":
		result = s.toolVerify(params.Arguments)
	case "bundle":
		result = s.toolBundle(params.Arguments)
	case "status":
		result = s.toolStatus()
	default:
		result = newToolResponse(fmt.Sprintf("unknown tool: %s", params.Name), true)
	}

	return s.respond(id, result)
}

func (s *Server) toolStatus() toolResponse {
	return newToolResponse(fmt.Sprintf("ATB v%s \u2014 operational", s.version), false)
}

func (s *Server) toolVerify(raw json.RawMessage) toolResponse {
	var args verifyArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return newToolResponse(fmt.Sprintf("invalid verify arguments: %v", err), true)
	}

	cmdArgs := []string{"verify", args.Path}
	if args.Strict {
		cmdArgs = append(cmdArgs, "--strict")
	}

	out, err := exec.CommandContext(s.commandContext(), os.Args[0], cmdArgs...).CombinedOutput()
	if err != nil {
		return newToolResponse(string(out), true)
	}

	return newToolResponse(string(out), false)
}

func (s *Server) toolBundle(raw json.RawMessage) toolResponse {
	var args bundleArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return newToolResponse(fmt.Sprintf("invalid bundle arguments: %v", err), true)
	}

	out, err := exec.CommandContext(
		s.commandContext(),
		os.Args[0],
		"bundle",
		args.Source,
		"--output",
		args.Output,
	).CombinedOutput()
	if err != nil {
		return newToolResponse(string(out), true)
	}

	return newToolResponse(string(out), false)
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
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      rawMessageValue(id),
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
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
