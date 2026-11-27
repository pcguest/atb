// SPDX-License-Identifier: MIT
package jcs

import (
	"encoding/json"
	"testing"
)

// TestLegacyMarshalRawPinsRFC8785 pins the legacy wrapper to RFC 8785
// behaviour with a vector exercising key ordering, number serialisation, and
// string escaping. It exercises legacy MarshalRaw, not the strict entry point.
func TestLegacyMarshalRawPinsRFC8785(t *testing.T) {
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

func TestMarshalRawRFC8785(t *testing.T) {
	t.Run("canonicalises valid raw json", func(t *testing.T) {
		in := []byte(`{"b": 2, "a": {"y": 1e1, "x": "é"}, "c": [true, null]}`)
		want := `{"a":{"x":"é","y":10},"b":2,"c":[true,null]}`
		got, err := MarshalRawRFC8785(in)
		if err != nil {
			t.Fatalf("MarshalRawRFC8785: %v", err)
		}
		if string(got) != want {
			t.Fatalf("MarshalRawRFC8785 = %s, want %s", got, want)
		}
	})

	t.Run("enforces safe integer policy", func(t *testing.T) {
		_, err := MarshalRawRFC8785([]byte(`{"n":9007199254740993}`))
		if got := StrictErrorClass(err); got != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", got, err)
		}
	})

	t.Run("rejects invalid utf8", func(t *testing.T) {
		in := []byte{'"', 'a', 0xff, 'b', '"'}
		_, err := MarshalRawRFC8785(in)
		if got := StrictErrorClass(err); got != "invalid_utf8" {
			t.Fatalf("StrictErrorClass = %q, want invalid_utf8 (%v)", got, err)
		}
	})
}

func TestMarshalRFC8785(t *testing.T) {
	t.Run("accepts finite e20", func(t *testing.T) {
		got, err := MarshalRFC8785(1e20)
		if err != nil {
			t.Fatalf("MarshalRFC8785(1e20): %v", err)
		}
		if want := "100000000000000000000"; string(got) != want {
			t.Fatalf("MarshalRFC8785(1e20) = %s, want %s", got, want)
		}
	})

	t.Run("enforces json.Number policy", func(t *testing.T) {
		_, err := MarshalRFC8785(json.Number("9007199254740993"))
		if got := StrictErrorClass(err); got != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", got, err)
		}
	})

	t.Run("enforces nested json.Number policy", func(t *testing.T) {
		value := struct {
			N json.Number `json:"n"`
		}{N: json.Number("9007199254740993")}
		_, err := MarshalRFC8785(value)
		if got := StrictErrorClass(err); got != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", got, err)
		}
	})

	t.Run("enforces raw message policy", func(t *testing.T) {
		_, err := MarshalRFC8785(json.RawMessage(`{"n":9007199254740993}`))
		if got := StrictErrorClass(err); got != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", got, err)
		}
	})

	t.Run("rejects invalid utf8", func(t *testing.T) {
		_, err := MarshalRFC8785("a\xffb")
		if got := StrictErrorClass(err); got != "invalid_utf8" {
			t.Fatalf("StrictErrorClass = %q, want invalid_utf8 (%v)", got, err)
		}
	})

	t.Run("accepts json.Number e20", func(t *testing.T) {
		got, err := MarshalRFC8785(json.Number("1e20"))
		if err != nil {
			t.Fatalf("MarshalRFC8785(json.Number(1e20)): %v", err)
		}
		if want := "100000000000000000000"; string(got) != want {
			t.Fatalf("MarshalRFC8785(json.Number(1e20)) = %s, want %s", got, want)
		}
	})
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
