// SPDX-License-Identifier: MIT

// This file is the executable conformance runner for the authored step-1
// strict-profile vectors. It does not change production dispatch: it exercises
// only the new, non-production strict entry points.
package jcs_test

import (
	"bytes"
	"testing"

	"github.com/pcguest/atb/v2/pkg/jcs"
)

func TestStrictPositiveConformance(t *testing.T) {
	file := loadJSON[positiveFile](t, positiveVectorsFile)
	if len(file.Cases) == 0 {
		t.Fatal("positive vector file has no cases")
	}

	for _, tc := range file.Cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)
			want := mustHex(t, tc.CanonicalUTF8Hex)

			got, err := jcs.MarshalRawRFC8785(input)
			if err != nil {
				t.Fatalf("MarshalRawRFC8785(%x): %v", input, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("MarshalRawRFC8785(%x) = %x, want %x", input, got, want)
			}
		})
	}
}

func TestStrictNegativeConformance(t *testing.T) {
	file := loadJSON[negativeFile](t, negativeVectorsFile)
	if len(file.Cases) == 0 {
		t.Fatal("negative vector file has no cases")
	}

	for _, tc := range file.Cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)

			got, err := jcs.MarshalRawRFC8785(input)
			if err == nil {
				t.Fatalf("MarshalRawRFC8785(%x) = %x, want error class %q", input, got, tc.ExpectedError)
			}
			if class := jcs.StrictErrorClass(err); class != tc.ExpectedError {
				t.Fatalf("MarshalRawRFC8785(%x) error class = %q, want %q (%v)", input, class, tc.ExpectedError, err)
			}
		})
	}
}
