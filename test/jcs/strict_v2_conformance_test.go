// SPDX-License-Identifier: MIT

// This file is the executable conformance runner for the authored
// atb-jcs-rfc8785-2 vectors. It exercises only non-production entry points and
// does not change hashing, bundle, or verifier dispatch.
package jcs_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/pcguest/atb/v2/pkg/jcs"
)

const v2PositiveVectorsFile = "strict_v2_positive_vectors.json"
const v2NegativeVectorsFile = "strict_v2_negative_vectors.json"
const numericParityCasesFile = "numeric_parity_cases.json"

const v2ProfileID = "atb-jcs-rfc8785-2"
const numericParitySchema = "atb.jcs.numeric-parity-cases.v1"
const numericParityProfileID = "atb-jcs-rfc8785-1-and-rfc8785-2"

type v2PositiveCase struct {
	ID string `json:"id"`

	Category string `json:"category"`

	InputUTF8Hex string `json:"input_utf8_hex"`

	CanonicalUTF8Hex string `json:"canonical_utf8_hex"`

	LegacyDivergence divergence `json:"legacy_divergence"`

	Notes string `json:"notes"`
}

type v2PositiveFile struct {
	Schema string `json:"schema"`

	Profile string `json:"profile"`

	Status string `json:"status"`

	Cases []v2PositiveCase `json:"cases"`
}

type v2NegativeCase struct {
	ID string `json:"id"`

	Category string `json:"category"`

	InputUTF8Hex string `json:"input_utf8_hex"`

	ExpectedError string `json:"expected_error"`

	Notes string `json:"notes"`
}

type v2NegativeFile struct {
	Schema string `json:"schema"`

	Profile string `json:"profile"`

	Status string `json:"status"`

	Cases []v2NegativeCase `json:"cases"`
}

type numericParityExpected struct {
	Go string `json:"go"`

	Python string `json:"python"`

	TypeScript string `json:"typescript"`
}

type numericParityCase struct {
	ID string `json:"id"`

	Profile string `json:"profile"`

	InputUTF8Hex string `json:"input_utf8_hex"`

	LegacyExpected numericParityExpected `json:"legacy_expected"`

	StrictExpected numericParityExpected `json:"strict_expected"`

	Notes string `json:"notes"`
}

type numericParityFile struct {
	Schema string `json:"schema"`

	Profile string `json:"profile"`

	Status string `json:"status"`

	Cases []numericParityCase `json:"cases"`
}

func TestStrictV2FixtureMetadata(t *testing.T) {
	positive := loadJSON[v2PositiveFile](t, v2PositiveVectorsFile)
	if positive.Schema != positiveSchema {
		t.Fatalf("positive schema = %q, want %q", positive.Schema, positiveSchema)
	}
	if positive.Profile != v2ProfileID {
		t.Fatalf("positive profile = %q, want %q", positive.Profile, v2ProfileID)
	}
	if positive.Status != executableStatus {
		t.Fatalf("positive status = %q, want %q", positive.Status, executableStatus)
	}
	if got := len(positive.Cases); got != 24 {
		t.Fatalf("positive case count = %d, want 24", got)
	}

	negative := loadJSON[v2NegativeFile](t, v2NegativeVectorsFile)
	if negative.Schema != negativeSchema {
		t.Fatalf("negative schema = %q, want %q", negative.Schema, negativeSchema)
	}
	if negative.Profile != v2ProfileID {
		t.Fatalf("negative profile = %q, want %q", negative.Profile, v2ProfileID)
	}
	if negative.Status != executableStatus {
		t.Fatalf("negative status = %q, want %q", negative.Status, executableStatus)
	}
	if got := len(negative.Cases); got != 9 {
		t.Fatalf("negative case count = %d, want 9", got)
	}

	parity := loadJSON[numericParityFile](t, numericParityCasesFile)
	if parity.Schema != numericParitySchema {
		t.Fatalf("numeric parity schema = %q, want %q", parity.Schema, numericParitySchema)
	}
	if parity.Profile != numericParityProfileID {
		t.Fatalf("numeric parity profile = %q, want %q", parity.Profile, numericParityProfileID)
	}
	if parity.Status != frozenStatus {
		t.Fatalf("numeric parity status = %q, want %q", parity.Status, frozenStatus)
	}
	if len(parity.Cases) != 1 {
		t.Fatalf("numeric parity case count = %d, want 1", len(parity.Cases))
	}

	seen := map[string]string{}
	addIDs := func(file string, cases []string) {
		for _, id := range cases {
			if previous, ok := seen[id]; ok {
				t.Fatalf("v2 case id %q appears in both %s and %s", id, previous, file)
			}
			seen[id] = file
		}
	}
	positiveIDs := make([]string, 0, len(positive.Cases))
	for _, tc := range positive.Cases {
		positiveIDs = append(positiveIDs, tc.ID)
	}
	negativeIDs := make([]string, 0, len(negative.Cases))
	for _, tc := range negative.Cases {
		negativeIDs = append(negativeIDs, tc.ID)
	}
	parityIDs := make([]string, 0, len(parity.Cases))
	for _, tc := range parity.Cases {
		parityIDs = append(parityIDs, tc.ID)
	}
	addIDs(v2PositiveVectorsFile, positiveIDs)
	addIDs(v2NegativeVectorsFile, negativeIDs)
	addIDs(numericParityCasesFile, parityIDs)
}

func TestStrictV2PositiveConformance(t *testing.T) {
	file := loadJSON[v2PositiveFile](t, v2PositiveVectorsFile)
	for _, tc := range file.Cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)
			want := mustHex(t, tc.CanonicalUTF8Hex)

			got, err := jcs.MarshalRawRFC8785V2(input)
			if err != nil {
				t.Fatalf("MarshalRawRFC8785V2(%x): %v", input, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("MarshalRawRFC8785V2(%x) = %x, want %x", input, got, want)
			}

			roundTrip, err := jcs.MarshalRawRFC8785V2(got)
			if err != nil {
				t.Fatalf("round-trip MarshalRawRFC8785V2(%x): %v", got, err)
			}
			if !bytes.Equal(roundTrip, got) {
				t.Fatalf("round-trip output = %x, want %x", roundTrip, got)
			}
		})
	}
}

func TestStrictV2NegativeConformance(t *testing.T) {
	file := loadJSON[v2NegativeFile](t, v2NegativeVectorsFile)
	for _, tc := range file.Cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)
			got, err := jcs.MarshalRawRFC8785V2(input)
			if err == nil {
				t.Fatalf("MarshalRawRFC8785V2(%x) = %x, want error class %q", input, got, tc.ExpectedError)
			}
			if class := jcs.StrictErrorClass(err); class != tc.ExpectedError {
				t.Fatalf("MarshalRawRFC8785V2(%x) error class = %q, want %q (%v)", input, class, tc.ExpectedError, err)
			}
		})
	}
}

func TestV1NegativeVectorsRemainRejectedByV2(t *testing.T) {
	file := loadJSON[negativeFile](t, negativeVectorsFile)
	for _, tc := range file.Cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)
			got, err := jcs.MarshalRawRFC8785V2(input)
			if err == nil {
				t.Fatalf("MarshalRawRFC8785V2(%x) = %x, want error class %q", input, got, tc.ExpectedError)
			}
			if class := jcs.StrictErrorClass(err); class != tc.ExpectedError {
				t.Fatalf("MarshalRawRFC8785V2(%x) error class = %q, want %q (%v)", input, class, tc.ExpectedError, err)
			}
		})
	}
}

func TestStrictV2ValueModePolicy(t *testing.T) {
	t.Run("1e20 rejected", func(t *testing.T) {
		got, err := jcs.MarshalRFC8785V2(1e20)
		if err == nil {
			t.Fatalf("MarshalRFC8785V2(1e20) = %s, want number_out_of_policy", got)
		}
		if class := jcs.StrictErrorClass(err); class != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", class, err)
		}
	})

	t.Run("1e21 accepted", func(t *testing.T) {
		got, err := jcs.MarshalRFC8785V2(1e21)
		if err != nil {
			t.Fatalf("MarshalRFC8785V2(1e21): %v", err)
		}
		if want := "1e+21"; string(got) != want {
			t.Fatalf("MarshalRFC8785V2(1e21) = %s, want %s", got, want)
		}
	})

	t.Run("1e-6 accepted with fixed output", func(t *testing.T) {
		got, err := jcs.MarshalRFC8785V2(1e-6)
		if err != nil {
			t.Fatalf("MarshalRFC8785V2(1e-6): %v", err)
		}
		if want := "0.000001"; string(got) != want {
			t.Fatalf("MarshalRFC8785V2(1e-6) = %s, want %s", got, want)
		}
	})

	t.Run("safe integer accepted", func(t *testing.T) {
		got, err := jcs.MarshalRFC8785V2(int64(9007199254740991))
		if err != nil {
			t.Fatalf("MarshalRFC8785V2(max safe integer): %v", err)
		}
		if want := "9007199254740991"; string(got) != want {
			t.Fatalf("MarshalRFC8785V2(max safe integer) = %s, want %s", got, want)
		}
	})

	t.Run("plain 10^21 value rejected", func(t *testing.T) {
		plain := new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)
		got, err := jcs.MarshalRFC8785V2(plain)
		if err == nil {
			t.Fatalf("MarshalRFC8785V2(10^21) = %s, want number_out_of_policy", got)
		}
		if class := jcs.StrictErrorClass(err); class != "number_out_of_policy" {
			t.Fatalf("StrictErrorClass = %q, want number_out_of_policy (%v)", class, err)
		}
	})
}

func TestNumericParityFixture(t *testing.T) {
	file := loadJSON[numericParityFile](t, numericParityCasesFile)
	for _, tc := range file.Cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)

			legacy, err := jcs.MarshalRaw(input)
			if err != nil {
				t.Fatalf("legacy MarshalRaw: %v", err)
			}
			if got := hexString(legacy); got != tc.LegacyExpected.Go {
				t.Fatalf("legacy Go output = %s, want %s", got, tc.LegacyExpected.Go)
			}

			strict, err := jcs.MarshalRawRFC8785V2(input)
			if err != nil {
				t.Fatalf("MarshalRawRFC8785V2: %v", err)
			}
			if got := hexString(strict); got != tc.StrictExpected.Go {
				t.Fatalf("strict Go output = %s, want %s", got, tc.StrictExpected.Go)
			}

			if tc.StrictExpected.Go != tc.StrictExpected.Python ||
				tc.StrictExpected.Go != tc.StrictExpected.TypeScript {
				t.Fatalf(
					"strict expected outputs must agree: go=%s python=%s typescript=%s",
					tc.StrictExpected.Go,
					tc.StrictExpected.Python,
					tc.StrictExpected.TypeScript,
				)
			}
		})
	}
}

func hexString(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = digits[c>>4]
		out[i*2+1] = digits[c&0x0f]
	}
	return string(out)
}
