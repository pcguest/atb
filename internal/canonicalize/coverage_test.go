// SPDX-License-Identifier: MIT
package canonicalize

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestMarshalAndStringEscapeContracts(t *testing.T) {
	got, err := Marshal(map[string]any{
		"z": "quote\" slash\\ back\b form\f line\n return\r tab\t control\u0001",
		"a": []any{nil, true, false, 1.5},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.HasPrefix(text, `{"a":[null,true,false,1.5],"z":"`) {
		t.Fatalf("canonical JSON=%q", text)
	}
	for _, escaped := range []string{`\"`, `\\`, `\b`, `\f`, `\n`, `\r`, `\t`, `\u0001`} {
		if !strings.Contains(text, escaped) {
			t.Fatalf("canonical JSON %q missing %q", text, escaped)
		}
	}
	if _, err := Marshal(func() {}); err == nil || !strings.Contains(err.Error(), "canonicalize: marshal") {
		t.Fatalf("unmarshalable value error=%v", err)
	}
	if _, err := MarshalRaw([]byte("{")); err == nil || !strings.Contains(err.Error(), "canonicalize: unmarshal") {
		t.Fatalf("invalid raw error=%v", err)
	}
}

func TestUnsupportedValueAndNumericBoundaries(t *testing.T) {
	var buf bytes.Buffer
	if err := writeValue(&buf, 1); err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("unsupported value error=%v", err)
	}
	if serializeNumber(math.NaN()) != "null" || serializeNumber(math.Inf(1)) != "null" {
		t.Fatal("non-finite values must serialize as null")
	}
	if !utf16Less("a", "aa") || utf16Less("aa", "a") || utf16Less("b", "a") {
		t.Fatal("UTF-16 ordering boundaries failed")
	}
}
