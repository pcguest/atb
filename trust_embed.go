package atbembed

import (
	"embed"
	"io/fs"
	"path"
)

// TrustEvidenceFS stores the shipped evidence files referenced by trust-report.
//
//go:embed SECURITY.md incident-response.md docs/spec-v1.0.md docs/security.md docs/quickstart.md docs/ai-integration.md schemas/event.v1.json cmd/atb/main_test.go test/golden/golden_test.go sdk/python/tests/test_properties.py
var TrustEvidenceFS embed.FS

// HasEmbeddedTrustEvidence reports whether the shipped binary contains the
// requested repo-relative trust evidence file.
func HasEmbeddedTrustEvidence(name string) bool {
	info, err := fs.Stat(TrustEvidenceFS, path.Clean(name))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
