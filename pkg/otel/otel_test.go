// SPDX-License-Identifier: MIT
package otel_test

import (
	"errors"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/pkg/otel"
)

func TestNewDefaultTranslator(t *testing.T) {
	t.Parallel()
	if got := otel.NewDefaultTranslator(); got == nil {
		t.Fatal("NewDefaultTranslator() returned nil")
	}
}

func TestTranslate_mapsLLMSpan(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 9, 9, 15, 2, 0, time.UTC)
	end := start.Add(1200 * time.Millisecond)

	got, err := otel.Translate(otel.OTelSpan{
		TraceID:   "0102030405060708090a0b0c0d0e0f10",
		SpanID:    "0102030405060708",
		Name:      "gen_ai.chat",
		StartTime: start,
		EndTime:   end,
		Attributes: map[string]any{
			"gen_ai.system":              "openai",
			"gen_ai.request.model":       "gpt-4.1-mini",
			"gen_ai.usage.input_tokens":  12,
			"gen_ai.usage.output_tokens": 28,
			"gen_ai.usage.total_tokens":  40,
			"privacy.mode":               "hash",
		},
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got.Type != event.TypeAILLMCall {
		t.Fatalf("Type = %q, want %q", got.Type, event.TypeAILLMCall)
	}
	if got.TraceID != "0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("TraceID = %q", got.TraceID)
	}
	data, ok := got.Data.(map[string]any)
	if !ok {
		t.Fatalf("Data type = %T, want map[string]any", got.Data)
	}
	if data["phase"] != "end" {
		t.Fatalf("phase = %v, want end", data["phase"])
	}
	context, ok := data["context"].(map[string]any)
	if !ok {
		t.Fatalf("context type = %T, want map[string]any", data["context"])
	}
	if context["provider"] != "openai" {
		t.Fatalf("provider = %v, want openai", context["provider"])
	}
	tokenUsage := context["token_usage"].(map[string]any)
	if tokenUsage["total_tokens"] != int64(40) {
		t.Fatalf("total_tokens = %v, want 40", tokenUsage["total_tokens"])
	}
}

func TestTranslate_returnsTypedErrorForUnmappableSpan(t *testing.T) {
	t.Parallel()
	_, err := otel.Translate(otel.OTelSpan{
		SpanID: "0102030405060708",
		Name:   "test.span",
	})
	if !errors.Is(err, otel.ErrUnmappableSpan) {
		t.Fatalf("Translate() error = %v, want %v", err, otel.ErrUnmappableSpan)
	}
}

func TestStubTransport_receive(t *testing.T) {
	t.Parallel()
	var transport otel.StubTransport
	err := transport.Receive(t.Context(), otel.OTelTrace{TraceID: "abc"})
	if !errors.Is(err, otel.ErrNotImplemented) {
		t.Fatalf("Receive() error = %v, want %v", err, otel.ErrNotImplemented)
	}
}

func TestReceiver_returnsUnmappableSpanError(t *testing.T) {
	t.Parallel()
	r := &otel.Receiver{
		Transport:  otel.StubTransport{},
		Translator: otel.DefaultTranslator{},
	}
	trace := otel.OTelTrace{
		TraceID: "0102030405060708090a0b0c0d0e0f10",
		Spans: []otel.OTelSpan{
			{TraceID: "0102030405060708090a0b0c0d0e0f10", SpanID: "0102030405060708", Name: "a"},
			{TraceID: "0102030405060708090a0b0c0d0e0f10", SpanID: "0908070605040302", Name: "b"},
		},
	}
	got, err := r.Receive(t.Context(), trace)
	if !errors.Is(err, otel.ErrUnmappableSpan) {
		t.Fatalf("Receive() error = %v, want %v", err, otel.ErrUnmappableSpan)
	}
	if got.SkippedCount != 0 {
		t.Fatalf("SkippedCount = %d, want 0", got.SkippedCount)
	}
}
