// SPDX-License-Identifier: MIT
package jcs

import "testing"

// TestMarshalRawRFC8785 pins the wrapper to RFC 8785 behaviour with a vector
// exercising key ordering, number serialisation, and string escaping.
func TestMarshalRawRFC8785(t *testing.T) {
	in := []byte(`{"b": 2, "a": {"y": 1e1, "x": "é"}, "c": [true, null]}`)
	want := `{"a":{"x":"é","y":10},"b":2,"c":[true,null]}`
	got, err := MarshalRaw(in)
	if err != nil {
		t.Fatalf("MarshalRaw: %v", err)
	}
	if string(got) != want {
		t.Fatalf("MarshalRaw = %s, want %s", got, want)
	}
}

func TestMarshalStruct(t *testing.T) {
	got, err := Marshal(struct {
		B int    `json:"b"`
		A string `json:"a"`
	}{B: 1, A: "x"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if want := `{"a":"x","b":1}`; string(got) != want {
		t.Fatalf("Marshal = %s, want %s", got, want)
	}
}
