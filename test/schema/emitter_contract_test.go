// SPDX-License-Identifier: MIT

// Package schema holds contract tests that pin the producer surfaces (SDK
// emitters and proxy lifecycle records) to the field contract declared in
// schemas/event.v1.json. They make required_fields and the documented optional
// fields executable: an emitter that drops a required field, or writes a field
// the schema does not declare, fails the build rather than drifting silently.
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/emit"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/proxy"
)

type documentedFieldSpec struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}

func loadDocumentedTypes(t *testing.T) map[string]documentedFieldSpec {
	t.Helper()
	schemaPath := filepath.Join("..", "..", "schemas", "event.v1.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema %q: %v", schemaPath, err)
	}
	var doc struct {
		DocumentedEventTypes map[string]json.RawMessage `json:"documented_event_types"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal schema %q: %v", schemaPath, err)
	}
	out := make(map[string]documentedFieldSpec, len(doc.DocumentedEventTypes))
	for typ, rawSpec := range doc.DocumentedEventTypes {
		if typ == "$comment" {
			continue
		}
		var spec documentedFieldSpec
		if err := json.Unmarshal(rawSpec, &spec); err != nil {
			t.Fatalf("unmarshal documented_event_types[%q]: %v", typ, err)
		}
		out[typ] = spec
	}
	return out
}

// assertPayloadMatchesSchema verifies every emitted key is declared in the
// schema's documented properties and every required field is present.
func assertPayloadMatchesSchema(t *testing.T, documented map[string]documentedFieldSpec, eventType string, data map[string]any) {
	t.Helper()
	spec, ok := documented[eventType]
	if !ok {
		t.Fatalf("event type %q has no documented_event_types entry in schema", eventType)
	}
	for key := range data {
		if _, declared := spec.Properties[key]; !declared {
			t.Errorf("%s: emitted undeclared field %q (add it to schemas/event.v1.json documented_event_types.properties)", eventType, key)
		}
	}
	for _, req := range spec.Required {
		if _, present := data[req]; !present {
			t.Errorf("%s: required field %q missing from emitted payload", eventType, req)
		}
	}
}

// TestGoEmittersHonourSchemaContract drives every oversight emitter with all
// optional fields populated and asserts the resulting payload matches the
// schema field contract.
func TestGoEmittersHonourSchemaContract(t *testing.T) {
	documented := loadDocumentedTypes(t)

	var captured []event.Event
	appendFn := func(ev event.Event) (string, error) {
		captured = append(captured, ev)
		return "record-hash", nil
	}
	em, err := emit.NewEmitter("session-1", appendFn)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}

	if err := em.ToolCall(emit.ToolCallOptions{
		ToolName:     "search",
		ActorID:      "actor-1",
		InputDigest:  "deadbeef",
		OutputDigest: "cafebabe",
	}); err != nil {
		t.Fatalf("ToolCall: %v", err)
	}
	if err := em.DataExport(emit.DataExportOptions{
		ExportTarget:   "s3://bucket/export",
		ActorID:        "actor-1",
		RecordCount:    5,
		Classification: "pii",
	}); err != nil {
		t.Fatalf("DataExport: %v", err)
	}
	if err := em.HumanOverride(emit.HumanOverrideOptions{
		OverrideReason:     "manual review",
		ActorID:            "actor-1",
		OverriddenActionID: "action-9",
	}); err != nil {
		t.Fatalf("HumanOverride: %v", err)
	}
	if err := em.HumanApproval(emit.HumanApprovalOptions{
		ApprovedActionID: "action-9",
		ActorID:          "actor-1",
		ApproverID:       "approver-2",
		Note:             "looks fine",
	}); err != nil {
		t.Fatalf("HumanApproval: %v", err)
	}

	if len(captured) != 4 {
		t.Fatalf("expected 4 emitted events, got %d", len(captured))
	}
	for _, ev := range captured {
		data, ok := ev.Data.(map[string]any)
		if !ok {
			t.Fatalf("%s: emitted Data is %T, want map[string]any", ev.Type, ev.Data)
		}
		assertPayloadMatchesSchema(t, documented, ev.Type, data)
	}
}

// TestProxyLifecycleRecordsHonourSchemaContract checks the proxy-internal
// lifecycle records (session.close, exchange.complete) against the schema,
// including the always-emitted and optional fields beyond the required set.
func TestProxyLifecycleRecordsHonourSchemaContract(t *testing.T) {
	documented := loadDocumentedTypes(t)

	sess := &proxy.Session{
		ID:            "sess-1",
		ActorID:       "actor-1",
		Model:         "gpt-4",
		ExchangeCount: 3,
		TotalTokens:   120,
	}

	closeRec := proxy.SessionCloseRecord(sess)
	assertPayloadMatchesSchema(t, documented, proxy.TypeSessionClose, closeRec)

	exchangeRec := proxy.ExchangeCompleteRecord(
		sess,
		"0001",
		"request-record-hash",
		"gpt-4",
		80, // prompt tokens
		40, // output tokens
		2,  // tool calls
		time.Now().UTC(),
	)
	assertPayloadMatchesSchema(t, documented, proxy.TypeExchangeComplete, exchangeRec)
}
