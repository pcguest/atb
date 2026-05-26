// SPDX-License-Identifier: MIT
package otel_test

import (
	"errors"
	"testing"

	"github.com/pcguest/atb/pkg/otel"
)

func TestTranslate_scaffoldReturnsNotImplemented(t *testing.T) {
	t.Parallel()
	_, err := otel.Translate(otel.OTelSpan{
		TraceID: "0102030405060708090a0b0c0d0e0f10",
		SpanID:  "0102030405060708",
		Name:    "test.span",
	})
	if !errors.Is(err, otel.ErrNotImplemented) {
		t.Fatalf("Translate() error = %v, want %v", err, otel.ErrNotImplemented)
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

func TestReceiver_collectsSkippedOnNotImplemented(t *testing.T) {
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
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}
	if got.SkippedCount != 2 {
		t.Fatalf("SkippedCount = %d, want 2", got.SkippedCount)
	}
	if len(got.Events) != 0 {
		t.Fatalf("Events len = %d, want 0", len(got.Events))
	}
}
