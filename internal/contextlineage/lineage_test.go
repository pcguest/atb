// SPDX-License-Identifier: MIT
package contextlineage

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func TestBuildKeepsContextOperationsDistinct(t *testing.T) {
	t.Parallel()

	records := []bundle.Record{
		{Event: hash.Event{Sequence: 1, Type: "ai.context.unit", Data: map[string]any{
			"unit_id": "source", "kind": "retrieved_passage", "digest": "aaa",
		}}},
		{Event: hash.Event{Sequence: 2, Type: "ai.context.unit", Data: map[string]any{
			"unit_id": "summary", "kind": "compacted_summary", "digest": "bbb",
		}}},
		{Event: hash.Event{Sequence: 3, Type: "ai.context.operation", Data: map[string]any{
			"operation_id": "compact-1", "operation": "compact",
			"input_unit_ids": []any{"source"}, "output_unit_ids": []any{"summary"},
			"input_token_count": float64(100), "output_token_count": float64(30),
		}}},
		{Event: hash.Event{Sequence: 4, Type: "ai.context.operation", Data: map[string]any{
			"operation_id": "assemble-1", "operation": "assemble",
			"input_unit_ids": []any{"summary"}, "output_unit_ids": []any{},
		}}},
	}

	lineage := Build(records)
	if len(lineage.Units) != 2 || len(lineage.Operations) != 2 {
		t.Fatalf("lineage = %#v", lineage)
	}
	if lineage.Operations[0].Type != "compact" || lineage.Operations[1].Type != "assemble" {
		t.Fatalf("operations = %#v", lineage.Operations)
	}
	if len(lineage.Warnings) != 0 {
		t.Fatalf("warnings = %v", lineage.Warnings)
	}
}

func TestBuildReportsUnavailableReferencesWithoutInventingUnits(t *testing.T) {
	t.Parallel()

	records := []bundle.Record{{Event: hash.Event{Sequence: 7, Type: "ai.context.operation", Data: map[string]any{
		"operation_id": "cache-1", "operation": "cache_reuse",
		"input_unit_ids": []any{"missing"}, "output_unit_ids": []any{},
	}}}}

	lineage := Build(records)
	if len(lineage.Units) != 0 || len(lineage.Warnings) != 1 {
		t.Fatalf("lineage = %#v", lineage)
	}
}
