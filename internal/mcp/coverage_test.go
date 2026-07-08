// SPDX-License-Identifier: MIT
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type testContextKey struct{}

func TestProtocolValidationAndNotifications(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		code int
	}{
		{name: "parse error", raw: "{", code: -32700},
		{name: "invalid version", raw: `{"jsonrpc":"1.0","id":1,"method":"tools/list"}`, code: -32600},
		{name: "invalid initialize params", raw: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":"bad"}`, code: -32602},
		{name: "invalid call params", raw: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":"bad"}`, code: -32602},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := runServer(t, tc.raw+"\n")
			if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != tc.code {
				t.Fatalf("responses=%+v", responses)
			}
		})
	}
	if responses := runServer(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"); len(responses) != 0 {
		t.Fatalf("notification responses=%+v", responses)
	}

	var output strings.Builder
	srv := New("1.2.3", strings.NewReader("\n"), &output)
	if srv == nil || srv.version != "1.2.3" {
		t.Fatalf("server=%+v", srv)
	}
	//lint:ignore SA1012 Serve explicitly accepts nil and substitutes a background context.
	if err := srv.Serve(nil); err != nil {
		t.Fatalf("Serve(nil): %v", err)
	}
}

func TestMCPArgumentHelpers(t *testing.T) {
	if got, err := decodeObjectArguments(nil); err != nil || len(got) != 0 {
		t.Fatalf("empty arguments=%v err=%v", got, err)
	}
	if got, err := decodeObjectArguments(json.RawMessage("null")); err != nil || len(got) != 0 {
		t.Fatalf("null arguments=%v err=%v", got, err)
	}
	args, err := decodeObjectArguments(json.RawMessage(`{"name":" value ","count":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeObjectArguments(json.RawMessage("{")); err == nil {
		t.Fatal("invalid JSON arguments succeeded")
	}
	if err := rejectUnknownFields(args, map[string]struct{}{"name": {}, "count": {}}); err != nil {
		t.Fatalf("allowed fields: %v", err)
	}
	if err := rejectUnknownFields(args, map[string]struct{}{"name": {}}); err == nil {
		t.Fatal("unknown field accepted")
	}
	if got, err := requireStringField(args, "name"); err != nil || got != "value" {
		t.Fatalf("required string=%q err=%v", got, err)
	}
	for _, tc := range []map[string]any{{}, {"name": 1}, {"name": " "}} {
		if _, err := requireStringField(tc, "name"); err == nil {
			t.Fatalf("required string accepted %#v", tc)
		}
	}
	if got, ok, err := optionalStringField(args, "missing"); err != nil || ok || got != "" {
		t.Fatalf("missing optional=%q,%v err=%v", got, ok, err)
	}
	if got, ok, err := optionalStringField(args, "name"); err != nil || !ok || got != " value " {
		t.Fatalf("optional=%q,%v err=%v", got, ok, err)
	}
	if _, _, err := optionalStringField(map[string]any{"name": 1}, "name"); err == nil {
		t.Fatal("non-string optional accepted")
	}

	integerCases := []struct {
		value any
		want  int
		ok    bool
	}{
		{value: json.Number("2"), want: 2, ok: true},
		{value: json.Number("2.5"), ok: false},
		{value: float64(3), want: 3, ok: true},
		{value: 3.5, ok: false},
		{value: "3", ok: false},
	}
	for _, tc := range integerCases {
		got, err := requireIntegerField(map[string]any{"count": tc.value}, "count")
		if tc.ok && (err != nil || got != tc.want) {
			t.Fatalf("integer %#v=%d err=%v", tc.value, got, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("integer %#v accepted", tc.value)
		}
	}
	if _, err := requireIntegerField(nil, "count"); err == nil {
		t.Fatal("missing integer accepted")
	}
}

func TestMCPErrorAndResponseHelpers(t *testing.T) {
	for _, tc := range []struct {
		exitCode int
		code     int
		message  string
	}{
		{exitCode: 1, code: -32602, message: "invalid params"},
		{exitCode: 2, code: -32001, message: "integrity verification failure"},
		{exitCode: 3, code: -32002, message: "system error"},
		{exitCode: 99, code: -32002, message: "system error"},
	} {
		var callErr *callError
		err := mcpError(tc.exitCode, " details ")
		if !errors.As(err, &callErr) || callErr.Code != tc.code || callErr.Message != tc.message {
			t.Fatalf("mcpError(%d)=%+v", tc.exitCode, err)
		}
		if callErr.Error() != tc.message {
			t.Fatalf("Error()=%q", callErr.Error())
		}
	}

	if abbreviateHash("short") != "short" || abbreviateHash(strings.Repeat("a", 20)) != strings.Repeat("a", 16)+"..." {
		t.Fatal("hash abbreviation failed")
	}
	id := json.RawMessage(`"request-1"`)
	if rawMessageValue(nil) != nil || rawMessageValue(&id) != "request-1" {
		t.Fatal("raw message conversion failed")
	}
	invalidID := json.RawMessage("{")
	if rawMessageValue(&invalidID) != nil {
		t.Fatal("invalid raw message should be nil")
	}

	if _, err := newStructuredToolResponse(func() {}); err == nil {
		t.Fatal("unmarshalable structured response succeeded")
	}
	ctx := context.WithValue(context.Background(), testContextKey{}, "value")
	srv := &Server{ctx: ctx}
	if srv.commandContext() != ctx {
		t.Fatal("server context not returned")
	}
	if (&Server{}).commandContext() == nil {
		t.Fatal("background context is nil")
	}

	var output bytes.Buffer
	srv = &Server{out: bufio.NewWriter(&output)}
	if err := srv.respondCallError(&id, errors.New("plain")); err != nil {
		t.Fatalf("respond plain error: %v", err)
	}
	if err := srv.respondCallError(&id, mcpError(1, "bad field")); err != nil {
		t.Fatalf("respond call error: %v", err)
	}
	if err := srv.out.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "system error") || !strings.Contains(output.String(), "invalid params") {
		t.Fatalf("responses=%q", output.String())
	}

	srv = &Server{out: bufio.NewWriter(failingWriter{err: errors.New("write failed")})}
	if err := srv.respondError(nil, -32002, "system error"); err == nil {
		t.Fatal("expected response writer error")
	}
}
