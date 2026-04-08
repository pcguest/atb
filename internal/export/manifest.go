package export

import "github.com/pcguest/atb/internal/verify"

type ComplianceManifest struct {
	BundlePath         string                 `json:"bundle_path"`
	ExportFormat       string                 `json:"export_format"`
	GeneratedAt        string                 `json:"generated_at"`
	VerifyResult       *verify.VerifierReport `json:"verify_result,omitempty"`
	Files              []ComplianceFile       `json:"files"`
	RegulatoryCoverage []string               `json:"regulatory_coverage"`
}

type ComplianceFile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
}
