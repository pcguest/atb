// SPDX-License-Identifier: MIT
package custody_test

import (
	"encoding/json"
	"testing"

	"github.com/pcguest/atb/pkg/custody"
)

const wantVerifyReportSchemaSHA256 = "ed25fdbcaa7e811b13fb1d365530f1e2b460ffb3585a0053aea3e8b162d6d5af"

func TestVerifyReportSchemaFrozen(t *testing.T) {
	raw := custody.VerifyReportSchemaJSON()
	if len(raw) == 0 {
		t.Fatal("empty schema")
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if doc["$id"] == nil {
		t.Fatal("schema missing $id")
	}
	if custody.VerifyReportSchemaVersion == "" {
		t.Fatal("empty schema version")
	}
	if custody.VerifyReportSchemaSHA256() == "" {
		t.Fatal("empty schema hash")
	}
	if len(custody.VerifyReportSchemaSHA256()) != 64 {
		t.Fatalf("schema hash length = %d", len(custody.VerifyReportSchemaSHA256()))
	}
	if got := custody.VerifyReportSchemaSHA256(); got != wantVerifyReportSchemaSHA256 {
		t.Fatalf("schema hash = %s, want %s", got, wantVerifyReportSchemaSHA256)
	}
}
