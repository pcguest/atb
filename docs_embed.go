// SPDX-License-Identifier: MIT
package atbembed

import (
	"embed"
	"io/fs"
	"path"
	"sort"
)

// ExportDocsFS stores the docs bundled into export archives.
//
//go:embed docs/spec-v1.0.md docs/security.md docs/incident-response.md docs/compliance/*.md docs/compliance/pii-fields.json
var ExportDocsFS embed.FS

// ReadExportDoc returns an embedded export doc by repo-relative path.
func ReadExportDoc(name string) ([]byte, error) {
	return ExportDocsFS.ReadFile(path.Clean(name))
}

// ListComplianceDocs returns embedded compliance doc paths in stable order.
func ListComplianceDocs() ([]string, error) {
	entries, err := fs.ReadDir(ExportDocsFS, "docs/compliance")
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		paths = append(paths, path.Join("docs/compliance", entry.Name()))
	}
	sort.Strings(paths)
	return paths, nil
}

// ReadEmbeddedPIIFields returns the bundled default PII field rules.
func ReadEmbeddedPIIFields() ([]byte, error) {
	return ExportDocsFS.ReadFile("docs/compliance/pii-fields.json")
}
