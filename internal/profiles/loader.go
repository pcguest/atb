package profiles

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*.yaml
var templateFS embed.FS

var (
	loadTemplatesOnce sync.Once
	loadTemplatesErr  error
	templateSchemas   map[string]ProfileSchema
)

func ParseSchema(data []byte) (ProfileSchema, error) {
	var schema ProfileSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return ProfileSchema{}, fmt.Errorf("parse schema: %w", err)
	}
	schema = normaliseSchema(schema)
	if err := ValidateSchema(schema); err != nil {
		return ProfileSchema{}, err
	}
	return schema, nil
}

// LoadSchema returns the ProfileSchema for id, or an error if id is unknown
// or the embedded templates failed to load.
func LoadSchema(id string) (ProfileSchema, error) {
	loadTemplatesOnce.Do(loadTemplates)
	if loadTemplatesErr != nil {
		return ProfileSchema{}, loadTemplatesErr
	}
	schema, ok := templateSchemas[id]
	if !ok {
		return ProfileSchema{}, fmt.Errorf("unknown profile schema %q", id)
	}
	return cloneSchema(schema), nil
}

func MustLoadSchema(id string) ProfileSchema {
	schema, err := LoadSchema(id)
	if err != nil {
		panic(err)
	}
	return schema
}

func HasSchema(id string) bool {
	loadTemplatesOnce.Do(loadTemplates)
	if loadTemplatesErr != nil {
		return false
	}
	_, ok := templateSchemas[id]
	return ok
}

func loadTemplates() {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		loadTemplatesErr = fmt.Errorf("read embedded profile templates: %w", err)
		return
	}

	loaded := make(map[string]ProfileSchema, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".yaml" {
			continue
		}

		content, err := templateFS.ReadFile(path.Join("templates", entry.Name()))
		if err != nil {
			loadTemplatesErr = fmt.Errorf("read embedded profile template %q: %w", entry.Name(), err)
			return
		}

		schema, err := ParseSchema(content)
		if err != nil {
			loadTemplatesErr = fmt.Errorf("load embedded profile template %q: %w", entry.Name(), err)
			return
		}
		if _, ok := loaded[schema.ID]; ok {
			loadTemplatesErr = fmt.Errorf("duplicate embedded profile schema id %q", schema.ID)
			return
		}
		loaded[schema.ID] = schema
	}

	templateSchemas = loaded
}

func normaliseSchema(schema ProfileSchema) ProfileSchema {
	schema.ID = strings.TrimSpace(schema.ID)
	schema.WorkflowClass = strings.TrimSpace(schema.WorkflowClass)
	schema.SCMode = strings.TrimSpace(schema.SCMode)

	schema.BlindSpots = trimAndFilter(schema.BlindSpots)

	normalisedWeights := make(map[string]float64, len(schema.Weights))
	for key, value := range schema.Weights {
		normalisedWeights[strings.TrimSpace(key)] = value
	}
	schema.Weights = normalisedWeights

	for i := range schema.Required {
		schema.Required[i] = normaliseEventRule(schema.Required[i])
	}
	for i := range schema.Optional {
		schema.Optional[i] = normaliseEventRule(schema.Optional[i])
	}
	for i := range schema.Relations {
		schema.Relations[i] = normaliseRelationRule(schema.Relations[i])
	}

	return schema
}

func normaliseEventRule(rule EventRule) EventRule {
	rule.Type = strings.TrimSpace(rule.Type)
	rule.Fields = trimAndFilter(rule.Fields)
	rule.Message = strings.TrimSpace(rule.Message)
	rule.Severity = strings.TrimSpace(rule.Severity)
	for i := range rule.RequiredWhen {
		rule.RequiredWhen[i].WhenType = strings.TrimSpace(rule.RequiredWhen[i].WhenType)
		rule.RequiredWhen[i].Message = strings.TrimSpace(rule.RequiredWhen[i].Message)
	}
	return rule
}

func normaliseRelationRule(rule RelationRule) RelationRule {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.From = strings.TrimSpace(rule.From)
	rule.To = strings.TrimSpace(rule.To)
	rule.Field = strings.TrimSpace(rule.Field)
	rule.Message = strings.TrimSpace(rule.Message)
	return rule
}

func trimAndFilter(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}

func cloneSchema(schema ProfileSchema) ProfileSchema {
	cloned := schema
	cloned.BlindSpots = append([]string(nil), schema.BlindSpots...)
	cloned.Weights = cloneWeights(schema.Weights)
	cloned.Required = append([]EventRule(nil), schema.Required...)
	cloned.Optional = append([]EventRule(nil), schema.Optional...)
	cloned.Relations = append([]RelationRule(nil), schema.Relations...)
	for i := range cloned.Required {
		cloned.Required[i].Fields = append([]string(nil), schema.Required[i].Fields...)
		cloned.Required[i].RequiredWhen = append([]RequiredWhenRule(nil), schema.Required[i].RequiredWhen...)
	}
	for i := range cloned.Optional {
		cloned.Optional[i].Fields = append([]string(nil), schema.Optional[i].Fields...)
		cloned.Optional[i].RequiredWhen = append([]RequiredWhenRule(nil), schema.Optional[i].RequiredWhen...)
	}
	return cloned
}

func cloneWeights(weights map[string]float64) map[string]float64 {
	cloned := make(map[string]float64, len(weights))
	for key, value := range weights {
		cloned[key] = value
	}
	return cloned
}

func templateFileNames() []string {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".yaml" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}
