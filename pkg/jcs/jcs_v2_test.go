// SPDX-License-Identifier: MIT
package jcs

import (
	"testing"
)

func TestV1AndV2EntryPointsAreDistinct(t *testing.T) {
	t.Run("v1 raw accepts 1e20 with frozen bytes", func(t *testing.T) {
		got, err := MarshalRawRFC8785([]byte(`{"n":1e20}`))
		if err != nil {
			t.Fatalf("MarshalRawRFC8785: %v", err)
		}
		if want := `{"n":100000000000000000000}`; string(got) != want {
			t.Fatalf("v1 raw = %s, want %s", got, want)
		}
	})

	t.Run("v2 raw rejects 1e20", func(t *testing.T) {
		got, err := MarshalRawRFC8785V2([]byte(`{"n":1e20}`))
		if err == nil {
			t.Fatalf("MarshalRawRFC8785V2 = %s, want number_out_of_policy", got)
		}
		if class := StrictErrorClass(err); class != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", class, err)
		}
	})

	t.Run("v1 value accepts 1e20 with frozen bytes", func(t *testing.T) {
		got, err := MarshalRFC8785(1e20)
		if err != nil {
			t.Fatalf("MarshalRFC8785: %v", err)
		}
		if want := "100000000000000000000"; string(got) != want {
			t.Fatalf("v1 value = %s, want %s", got, want)
		}
	})

	t.Run("v2 value rejects 1e20", func(t *testing.T) {
		got, err := MarshalRFC8785V2(1e20)
		if err == nil {
			t.Fatalf("MarshalRFC8785V2 = %s, want number_out_of_policy", got)
		}
		if class := StrictErrorClass(err); class != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", class, err)
		}
	})
}

func TestV2Boundaries(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantClass string
	}{
		{name: "1e16", input: `{"n":1e16}`, wantClass: "number_out_of_policy"},
		{name: "2e16", input: `{"n":2e16}`, wantClass: "number_out_of_policy"},
		{name: "1e19", input: `{"n":1e19}`, wantClass: "number_out_of_policy"},
		{name: "2^53 fraction", input: `{"n":9007199254740992.0}`, wantClass: "number_out_of_policy"},
		{name: "2^55 fraction", input: `{"n":36028797018963968.0}`, wantClass: "number_out_of_policy"},
		{name: "negative 2^53 fraction", input: `{"n":-9007199254740992.0}`, wantClass: "number_out_of_policy"},
		{name: "2^60 fraction", input: `{"n":1152921504606846976.0}`, wantClass: "number_out_of_policy"},
		{name: "1e21 exponential", input: `{"n":1e21}`, want: `{"n":1e+21}`},
		{name: "1e-6 fixed", input: `{"n":1e-6}`, want: `{"n":0.000001}`},
		{name: "safe integer", input: `{"n":9007199254740991}`, want: `{"n":9007199254740991}`},
		{name: "plain 10^21", input: `{"n":1000000000000000000000}`, wantClass: "number_out_of_policy"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalRawRFC8785V2([]byte(tt.input))
			if tt.wantClass != "" {
				if err == nil {
					t.Fatalf("MarshalRawRFC8785V2(%s) = %s, want %s", tt.input, got, tt.wantClass)
				}
				if class := StrictErrorClass(err); class != tt.wantClass {
					t.Fatalf("StrictErrorClass = %q, want %q (%v)", class, tt.wantClass, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("MarshalRawRFC8785V2(%s): %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Fatalf("MarshalRawRFC8785V2(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}
