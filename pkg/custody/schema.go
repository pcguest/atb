// SPDX-License-Identifier: MIT
package custody

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
)

// VerifyReportSchemaVersion identifies the frozen JSON Schema for verify.report.v1.
const VerifyReportSchemaVersion = "verify.report.v1.schema.2"

//go:embed schema/verify.report.v1.schema.json
var verifyReportSchemaJSON []byte

// VerifyReportSchemaJSON returns the frozen JSON Schema bytes for verify.report.v1.
func VerifyReportSchemaJSON() []byte {
	out := make([]byte, len(verifyReportSchemaJSON))
	copy(out, verifyReportSchemaJSON)
	return out
}

// VerifyReportSchemaSHA256 returns the lowercase hex SHA-256 of the schema document.
func VerifyReportSchemaSHA256() string {
	sum := sha256.Sum256(verifyReportSchemaJSON)
	return hex.EncodeToString(sum[:])
}
