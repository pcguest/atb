package main

import (
	"bytes"
	"strings"
	"testing"

	profiledsl "github.com/pcguest/atb/internal/profiles"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestObligationSpecs_DeriveFromSchema(t *testing.T) {
	schema := profiledsl.MustLoadSchema("atb.profile.rag_answer")

	specs := obligationSpecs(schema.ID)
	if len(specs) != len(schema.Required)+len(schema.Optional) {
		t.Fatalf("unexpected obligation spec count: got %d want %d", len(specs), len(schema.Required)+len(schema.Optional))
	}
	if specs[0].eventType != schema.Required[0].Type {
		t.Fatalf("unexpected first obligation event type: got %q want %q", specs[0].eventType, schema.Required[0].Type)
	}

	foundWarning := false
	for _, spec := range specs {
		if spec.eventType == "ai.retrieval.executed" {
			foundWarning = spec.warning && strings.Contains(spec.message, "RAG answer provenance")
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected warning obligation derived from schema, got %+v", specs)
	}
}

func TestRelationSpecs_DeriveFromSchema(t *testing.T) {
	schema := profiledsl.MustLoadSchema("atb.profile.privileged_tool_action")

	specs := relationSpecs(schema.ID)
	if len(specs) != len(schema.Relations) {
		t.Fatalf("unexpected relation spec count: got %d want %d", len(specs), len(schema.Relations))
	}
	if specs[0].from != schema.Relations[0].From || specs[0].to != schema.Relations[0].To {
		t.Fatalf("unexpected first relation spec: got %+v want from=%q to=%q", specs[0], schema.Relations[0].From, schema.Relations[0].To)
	}
}

func TestRenderVerifyTerminalReport_PrintsAnchorWarnings(t *testing.T) {
	var output bytes.Buffer

	renderVerifyTerminalReport(&output, verifypkg.Report{
		BundlePath: "bundle.atb",
		Anchoring: verifypkg.AnchoringResult{
			AnchorPresent: true,
			Errors: []string{
				"anchor record at index 2: data is int, want string",
				"anchor record at index 1: invalid JSON payload: unexpected end of JSON input",
			},
		},
		ResidualRisk: verifypkg.ResidualRisk{Level: "High"},
	})

	rendered := output.String()
	if !strings.Contains(rendered, "  anchor warning: anchor record at index 2: data is int, want string\n") {
		t.Fatalf("expected first anchor warning in output, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "  anchor warning: anchor record at index 1: invalid JSON payload: unexpected end of JSON input\n") {
		t.Fatalf("expected second anchor warning in output, got:\n%s", rendered)
	}
}
