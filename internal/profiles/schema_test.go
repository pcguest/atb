package profiles

import "testing"

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
		Weights: map[string]float64{
			"EC": 0.20,
			"FC": 0.15,
			"RC": 0.20,
			"TC": 0.05,
			"SC": 0.10,
			"XC": 0.10,
			"AC": 0.10,
			"GC": 0.10,
		},
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
