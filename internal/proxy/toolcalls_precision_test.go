// SPDX-License-Identifier: MIT
package proxy

import (
	"strings"
	"testing"
)

// TestExtractToolCallsPreservesLargeIntegers proves tool arguments are digested
// without float64 rounding, so large integers in tool input are not silently
// mutated on the accountability path.
func TestExtractToolCallsPreservesLargeIntegers(t *testing.T) {
	body := []byte(`{"content":[{"type":"tool_use","name":"transfer","input":{"account_id":123456789012345678,"amount":9007199254740993}}]}`)
	calls := ExtractToolCalls(body)
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	args := string(calls[0].Arguments)
	if !strings.Contains(args, "123456789012345678") {
		t.Fatalf("large account id was mutated: %s", args)
	}
	if !strings.Contains(args, "9007199254740993") {
		t.Fatalf("large amount was mutated: %s", args)
	}
}

// TestToolInputDigestDistinguishesLargeIntegers proves two tool inputs that
// differ only in an integer above 2^53 produce different digests.
func TestToolInputDigestDistinguishesLargeIntegers(t *testing.T) {
	a := ExtractToolCalls([]byte(`{"content":[{"type":"tool_use","name":"t","input":{"id":9007199254740993}}]}`))
	b := ExtractToolCalls([]byte(`{"content":[{"type":"tool_use","name":"t","input":{"id":9007199254740992}}]}`))
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("calls = %d/%d, want 1/1", len(a), len(b))
	}
	if a[0].InputDigest() == b[0].InputDigest() {
		t.Fatalf("distinct large integers produced the same tool-input digest")
	}
}
