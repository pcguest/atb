// SPDX-License-Identifier: MIT
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

type eventSchema struct {
	DocumentedEventTypes map[string]json.RawMessage `json:"documented_event_types"`
	EventTypes           []registryEventType        `json:"event_types"`
}

type documentedEventType struct {
	Required []string `json:"required"`
}

type registryEventType struct {
	Type           string   `json:"type"`
	RequiredFields []string `json:"required_fields"`
}

func normaliseFields(fields []string) []string {
	if len(fields) == 0 {
		return []string{}
	}
	return fields
}

func TestEventRequiredFieldsConsistentAcrossSchemaSections(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "event.v1.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema %q: %v", schemaPath, err)
	}

	var schema eventSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal schema %q: %v", schemaPath, err)
	}

	registryByType := make(map[string][]string, len(schema.EventTypes))
	for _, entry := range schema.EventTypes {
		registryByType[entry.Type] = entry.RequiredFields
	}

	documentedByType := make(map[string]documentedEventType, len(schema.DocumentedEventTypes))
	for eventType, raw := range schema.DocumentedEventTypes {
		if eventType == "$comment" {
			continue
		}
		var documented documentedEventType
		if err := json.Unmarshal(raw, &documented); err != nil {
			t.Fatalf("unmarshal documented_event_types[%q]: %v", eventType, err)
		}
		documentedByType[eventType] = documented
	}

	allowedRegistryOnly := map[string]struct{}{
		"atb.bundle.manifest":  {},
		"atb.bundle.anchor":    {},
		"atb.bundle.signature": {},
		"atb.snapshot":         {},
	}

	var missingInRegistry []string
	for eventType := range documentedByType {
		if _, ok := registryByType[eventType]; !ok {
			missingInRegistry = append(missingInRegistry, eventType)
		}
	}

	var missingInDocumented []string
	for eventType := range registryByType {
		if _, ok := documentedByType[eventType]; ok {
			continue
		}
		if _, allowed := allowedRegistryOnly[eventType]; allowed {
			continue
		}
		missingInDocumented = append(missingInDocumented, eventType)
	}

	slices.Sort(missingInRegistry)
	slices.Sort(missingInDocumented)

	if len(missingInRegistry) > 0 {
		t.Fatalf("schema mismatch: event types in documented_event_types but missing in event_types: %v", missingInRegistry)
	}
	if len(missingInDocumented) > 0 {
		t.Fatalf("schema mismatch: event types in event_types but missing in documented_event_types: %v", missingInDocumented)
	}

	for eventType, documented := range documentedByType {
		registryRequired, ok := registryByType[eventType]
		if !ok {
			continue
		}
		documentedRequired := normaliseFields(documented.Required)
		registryRequired = normaliseFields(registryRequired)
		if !reflect.DeepEqual(documentedRequired, registryRequired) {
			t.Fatalf(
				"required field mismatch for %q: documented_event_types.required=%v event_types.required_fields=%v",
				eventType,
				documentedRequired,
				registryRequired,
			)
		}
	}
}
