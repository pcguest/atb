// SPDX-License-Identifier: MIT
package canonicalize

import "testing"

func FuzzMarshal(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"a":1}`),
		[]byte(`{"z":1,"a":2,"m":3}`),
		[]byte(`[1,2,3]`),
		[]byte(`"hello"`),
		[]byte(`null`),
		[]byte(`true`),
		[]byte(`1e21`),
		[]byte(`1e-6`),
		[]byte(`0`),
		[]byte(`{"nested":{"b":1,"a":2},"top":true}`),
		[]byte(`{"unicode":"\u0000\u001f"}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("MarshalRaw panicked: %v", r)
			}
		}()

		got, err := MarshalRaw(data)
		if err != nil {
			return
		}

		gotAgain, err := MarshalRaw(got)
		if err != nil {
			t.Fatalf("MarshalRaw on canonical output returned error: %v", err)
		}
		if string(gotAgain) != string(got) {
			t.Fatalf("MarshalRaw output not idempotent: got %q, want %q", gotAgain, got)
		}
	})
}
