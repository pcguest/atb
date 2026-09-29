// SPDX-License-Identifier: MIT
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

// envelopeSchema pins the parts of schemas/event.v1.json that describe the
// canonical (hashed) event envelope, as opposed to the per-type payloads in
// documented_event_types.
type envelopeSchema struct {
	AdditionalProperties bool                       `json:"additionalProperties"`
	Required             []string                   `json:"required"`
	Properties           map[string]json.RawMessage `json:"properties"`
	Defs                 map[string]json.RawMessage `json:"$defs"`
}

type objectDef struct {
	Required   []string                   `json:"required"`
	Properties map[string]json.RawMessage `json:"properties"`
}

func loadEnvelopeSchema(t *testing.T) envelopeSchema {
	t.Helper()
	schemaPath := filepath.Join("..", "..", "schemas", "event.v1.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema %q: %v", schemaPath, err)
	}
	var schema envelopeSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal schema %q: %v", schemaPath, err)
	}
	return schema
}

// structJSONFields returns the JSON property names of a struct's exported
// fields, split into all fields and those without omitempty.
func structJSONFields(t *testing.T, value any) (all, required []string) {
	t.Helper()
	typ := reflect.TypeOf(value)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("json")
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		all = append(all, name)
		if !strings.Contains(","+opts+",", ",omitempty,") {
			required = append(required, name)
		}
	}
	slices.Sort(all)
	slices.Sort(required)
	return all, required
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func normaliseSorted(values []string) []string {
	out := append([]string(nil), values...)
	slices.Sort(out)
	return out
}

// TestEnvelopeSchemaRepresentsRuntimeEventFields protects the invariant that
// every field participating in canonical serialisation of event.Event is
// declared in the normative envelope schema. It would have failed when
// `acquisition` was added to the runtime Event struct but omitted from the
// schema.
func TestEnvelopeSchemaRepresentsRuntimeEventFields(t *testing.T) {
	schema := loadEnvelopeSchema(t)

	if schema.AdditionalProperties {
		t.Error("envelope schema must set additionalProperties:false so undeclared canonical fields are visible")
	}

	all, required := structJSONFields(t, event.Event{})

	if got, want := keysOf(schema.Properties), all; !slices.Equal(got, want) {
		t.Errorf("envelope properties do not match runtime Event fields\n  schema:  %v\n  runtime: %v", got, want)
	}
	if got, want := normaliseSorted(schema.Required), required; !slices.Equal(got, want) {
		t.Errorf("envelope required fields do not match runtime Event fields without omitempty\n  schema:  %v\n  runtime: %v", got, want)
	}
}

// TestEnvelopeSchemaRepresentsAcquisitionStructs protects the acquisition
// representation specifically: the schema $defs must describe the runtime
// AcquisitionInfo and CheckpointInfo shapes.
func TestEnvelopeSchemaRepresentsAcquisitionStructs(t *testing.T) {
	schema := loadEnvelopeSchema(t)

	check := func(defName string, value any) {
		t.Helper()
		raw, ok := schema.Defs[defName]
		if !ok {
			t.Fatalf("envelope schema $defs is missing %q", defName)
		}
		var def objectDef
		if err := json.Unmarshal(raw, &def); err != nil {
			t.Fatalf("unmarshal $defs.%s: %v", defName, err)
		}
		all, required := structJSONFields(t, value)
		if got := keysOf(def.Properties); !slices.Equal(got, all) {
			t.Errorf("$defs.%s properties do not match runtime struct\n  schema:  %v\n  runtime: %v", defName, got, all)
		}
		if got := normaliseSorted(def.Required); !slices.Equal(got, required) {
			t.Errorf("$defs.%s required fields do not match runtime struct without omitempty\n  schema:  %v\n  runtime: %v", defName, got, required)
		}
	}

	check("acquisition", event.AcquisitionInfo{})
	check("acquisition_checkpoint", event.CheckpointInfo{})
}
