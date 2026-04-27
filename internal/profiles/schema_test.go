// SPDX-License-Identifier: MIT
package profiles

import "testing"

func validWeights() map[string]float64 {
	return map[string]float64{
		"EC": 0.20,
		"FC": 0.15,
		"RC": 0.20,
		"TC": 0.05,
		"SC": 0.10,
		"XC": 0.10,
		"AC": 0.10,
		"GC": 0.10,
	}
}

func TestValidateSchema_Valid(t *testing.T) {
	for _, name := range templateFileNames() {
		name := name
		t.Run(name, func(t *testing.T) {
			content, err := templateFS.ReadFile("templates/" + name)
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", name, err)
			}
			if _, err := ParseSchema(content); err != nil {
				t.Fatalf("ParseSchema(%q): %v", name, err)
			}
		})
	}
}

func TestValidateSchema_BadWeights(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights: map[string]float64{
			"EC": 0.20,
			"FC": 0.15,
			"RC": 0.20,
			"TC": 0.05,
			"SC": 0.10,
			"XC": 0.09,
			"AC": 0.10,
			"GC": 0.10,
		},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected weight validation error")
	}
}

func TestValidateSchema_UnknownSev(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Required: []EventRule{
			{
				Type:     "ai.request.received",
				Message:  "required",
				Severity: "info",
			},
		},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected severity validation error")
	}
}

func TestValidateSchema_RequiredWhenUnknownEvent(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:         "ai.response.sent",
				Severity:     "warning",
				RequiredWhen: []RequiredWhenRule{{WhenType: "ai.unknown"}},
			},
		},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected error for unknown required_when when_type")
	}
}

func TestValidateSchema_RequiredWhenOnRequiredRuleRejected(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Required: []EventRule{
			{
				Type:         "ai.request.received",
				Severity:     "critical",
				RequiredWhen: []RequiredWhenRule{{WhenType: "ai.response.sent"}},
			},
		},
		Optional: []EventRule{
			{Type: "ai.response.sent", Severity: "warning"},
		},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected error for required_when on required rule")
	}
}

func TestValidateSchema_RequiredWhenCycle(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Optional: []EventRule{
			{
				Type:     "A",
				Severity: "warning",
				RequiredWhen: []RequiredWhenRule{
					{WhenType: "B"},
				},
			},
			{
				Type:     "B",
				Severity: "warning",
				RequiredWhen: []RequiredWhenRule{
					{WhenType: "A"},
				},
			},
		},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected error for required_when cycle")
	}
}

func TestValidateSchema_ValidRequiredWhen(t *testing.T) {
	schema := ProfileSchema{
		ID:            "atb.profile.test",
		Version:       1,
		WorkflowClass: "test",
		Weights:       validWeights(),
		Required: []EventRule{
			{
				Type:     "ai.request.received",
				Severity: "critical",
			},
		},
		Optional: []EventRule{
			{
				Type:     "ai.response.sent",
				Severity: "warning",
				RequiredWhen: []RequiredWhenRule{
					{
						WhenType:  "ai.request.received",
						AtOrAfter: true,
						Message:   "ai.response.sent required after ai.request.received",
					},
				},
			},
		},
	}

	if err := ValidateSchema(schema); err != nil {
		t.Fatalf("expected valid required_when rule, got %v", err)
	}
}
