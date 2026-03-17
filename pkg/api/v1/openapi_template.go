package apiv1

import _ "embed"

// openAPITemplateYAML stores the canonical OpenAPI spec template for viewer APIs.
//
//go:embed openapi.template.yaml
var openAPITemplateYAML []byte

// OpenAPITemplate returns the OpenAPI YAML template bytes used by doc generation.
func OpenAPITemplate() []byte {
	out := make([]byte, len(openAPITemplateYAML))
	copy(out, openAPITemplateYAML)
	return out
}
