// SPDX-License-Identifier: MIT

// Package jcs_test exercises the canonicalisation-profile vectors in test/jcs.
// It validates the strict fixtures, executes them through the non-production
// strict conformance runner, and pins the frozen legacy behaviour. Production
// dispatch remains on the legacy profile.
package jcs_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"unicode/utf8"

	"github.com/pcguest/atb/v2/internal/canonicalize"
)

const positiveVectorsFile = "strict_positive_vectors.json"
const negativeVectorsFile = "strict_negative_vectors.json"
const legacyVectorsFile = "legacy_freeze_vectors.json"
const profileCasesFile = "profile_selection_cases.json"

const positiveSchema = "atb.jcs.strict-positive-vectors.v1"
const negativeSchema = "atb.jcs.strict-negative-vectors.v1"
const legacySchema = "atb.jcs.legacy-freeze-vectors.v1"
const selectionSchema = "atb.jcs.profile-selection-cases.v1"

const strictProfileID = "atb-jcs-rfc8785-1"
const legacyProfileID = "atb-jcs-legacy-1"

const executableStatus = "executable"
const frozenStatus = "frozen"
const selectionStatus = "specification-and-current-reader"

var allowedErrorClasses = []string{
	"duplicate_key",
	"invalid_utf8",
	"lone_surrogate",
	"trailing_data",
	"non_finite_number",
	"number_out_of_policy",
	"number_unrepresentable",
	"invalid_json",
	"depth_limit",
	"missing_profile",
	"unsupported_profile",
}

var requiredPositiveCategories = []string{
	"key_ordering",
	"string_escaping",
	"numbers",
	"nesting",
}

var requiredNegativeCategories = []string{
	"duplicate_key",
	"invalid_utf8",
	"lone_surrogate",
	"trailing_data",
	"non_finite_number",
	"number_out_of_policy",
	"number_unrepresentable",
	"invalid_json",
}

type divergence struct {
	Go bool `json:"go"`

	Python bool `json:"python"`

	TypeScript bool `json:"typescript"`
}

type positiveCase struct {
	ID string `json:"id"`

	Category string `json:"category"`

	InputUTF8Hex string `json:"input_utf8_hex"`

	CanonicalUTF8Hex string `json:"canonical_utf8_hex"`

	LegacyDivergence divergence `json:"legacy_divergence"`
}

type positiveFile struct {
	Schema string `json:"schema"`

	Profile string `json:"profile"`

	Status string `json:"status"`

	Cases []positiveCase `json:"cases"`
}

type negativeCase struct {
	ID string `json:"id"`

	Category string `json:"category"`

	InputUTF8Hex string `json:"input_utf8_hex"`

	ExpectedError string `json:"expected_error"`
}

type negativeFile struct {
	Schema string `json:"schema"`

	Profile string `json:"profile"`

	Status string `json:"status"`

	Cases []negativeCase `json:"cases"`
}

type legacyRawCase struct {
	ID string `json:"id"`

	InputUTF8Hex string `json:"input_utf8_hex"`

	GoCanonicalUTF8Hex string `json:"go_canonical_utf8_hex"`

	GoError string `json:"go_error"`
}

type legacyValueExpected struct {
	Go string `json:"go"`

	Python string `json:"python"`

	TypeScript string `json:"typescript"`
}

type legacyValueCase struct {
	ID string `json:"id"`

	ValueJSON string `json:"value_json"`

	Expected legacyValueExpected `json:"expected"`

	EqualAcrossLanguages bool `json:"equal_across_languages"`
}

type legacyFile struct {
	Schema string `json:"schema"`

	Profile string `json:"profile"`

	Status string `json:"status"`

	RawCases []legacyRawCase `json:"raw_cases"`

	ValueCases []legacyValueCase `json:"value_cases"`
}

type profileProfiles struct {
	Legacy string `json:"legacy"`

	Strict string `json:"strict"`
}

type profileCase struct {
	ID string `json:"id"`

	ManifestVersion int `json:"manifest_version"`

	CanonicalProfile string `json:"canonical_profile"`

	CurrentReader string `json:"current_reader"`

	StrictReader string `json:"strict_reader"`
}

type profileFile struct {
	Schema string `json:"schema"`

	Status string `json:"status"`

	Profiles profileProfiles `json:"profiles"`

	ManifestField string `json:"manifest_field"`

	ManifestVersionStrict int `json:"manifest_version_strict"`

	Cases []profileCase `json:"cases"`
}

func loadJSON[T any](t *testing.T, name string) T {
	t.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return value
}

func mustHex(t *testing.T, encoded string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("invalid hex %q: %v", encoded, err)
	}
	return decoded
}

func isAllowedErrorClass(name string) bool {
	for _, candidate := range allowedErrorClasses {
		if candidate == name {
			return true
		}
	}
	return false
}

func checkCategories(t *testing.T, seen map[string]bool, required []string) {
	t.Helper()
	for _, category := range required {
		if !seen[category] {
			t.Fatalf("required category %q is not covered", category)
		}
	}
}

func TestStrictPositiveVectors(t *testing.T) {
	file := loadJSON[positiveFile](t, positiveVectorsFile)
	if file.Schema != positiveSchema {
		t.Fatalf("schema = %q, want %q", file.Schema, positiveSchema)
	}
	if file.Profile != strictProfileID {
		t.Fatalf("profile = %q, want %q", file.Profile, strictProfileID)
	}
	if file.Status != executableStatus {
		t.Fatalf("status = %q, want %q", file.Status, executableStatus)
	}
	if len(file.Cases) == 0 {
		t.Fatal("positive vector file has no cases")
	}

	seenCategories := map[string]bool{}
	seenIDs := map[string]bool{}
	for _, tc := range file.Cases {
		tc := tc
		if seenIDs[tc.ID] {
			t.Fatalf("duplicate case id %q", tc.ID)
		}
		seenIDs[tc.ID] = true
		seenCategories[tc.Category] = true

		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)
			canonical := mustHex(t, tc.CanonicalUTF8Hex)

			if !utf8.Valid(input) {
				t.Fatal("input_utf8_hex is not valid UTF-8")
			}
			if !utf8.Valid(canonical) {
				t.Fatal("canonical_utf8_hex is not valid UTF-8")
			}
			if !json.Valid(input) {
				t.Fatal("input_utf8_hex is not valid JSON")
			}
			if !json.Valid(canonical) {
				t.Fatal("canonical_utf8_hex is not valid JSON")
			}

			legacy, err := canonicalize.MarshalRaw(input)
			if err != nil {
				t.Fatalf("legacy MarshalRaw: %v", err)
			}
			diverges := !bytes.Equal(legacy, canonical)
			if diverges != tc.LegacyDivergence.Go {
				t.Fatalf(
					"legacy_divergence.go = %v, but legacy output %s strict canonical: got %s want %s",
					tc.LegacyDivergence.Go,
					ternary(diverges, "differs from", "matches"),
					hex.EncodeToString(legacy),
					tc.CanonicalUTF8Hex,
				)
			}
		})
	}

	checkCategories(t, seenCategories, requiredPositiveCategories)
}

func TestStrictNegativeVectors(t *testing.T) {
	file := loadJSON[negativeFile](t, negativeVectorsFile)
	if file.Schema != negativeSchema {
		t.Fatalf("schema = %q, want %q", file.Schema, negativeSchema)
	}
	if file.Profile != strictProfileID {
		t.Fatalf("profile = %q, want %q", file.Profile, strictProfileID)
	}
	if file.Status != executableStatus {
		t.Fatalf("status = %q, want %q", file.Status, executableStatus)
	}
	if len(file.Cases) == 0 {
		t.Fatal("negative vector file has no cases")
	}

	seenCategories := map[string]bool{}
	seenIDs := map[string]bool{}
	for _, tc := range file.Cases {
		tc := tc
		if seenIDs[tc.ID] {
			t.Fatalf("duplicate case id %q", tc.ID)
		}
		seenIDs[tc.ID] = true
		seenCategories[tc.Category] = true

		t.Run(tc.ID, func(t *testing.T) {
			// This fixture-integrity test validates the payload bytes and the
			// declared rejection class. strict_conformance_test.go executes the
			// same cases through the strict parser and asserts the class.
			mustHex(t, tc.InputUTF8Hex)
			if !isAllowedErrorClass(tc.ExpectedError) {
				t.Fatalf("expected_error %q is not an allowed error class", tc.ExpectedError)
			}
			if tc.Category != tc.ExpectedError {
				t.Fatalf("category %q does not match expected_error %q", tc.Category, tc.ExpectedError)
			}
		})
	}

	checkCategories(t, seenCategories, requiredNegativeCategories)
}

func TestLegacyFreezeVectors(t *testing.T) {
	file := loadJSON[legacyFile](t, legacyVectorsFile)
	if file.Schema != legacySchema {
		t.Fatalf("schema = %q, want %q", file.Schema, legacySchema)
	}
	if file.Profile != legacyProfileID {
		t.Fatalf("profile = %q, want %q", file.Profile, legacyProfileID)
	}
	if file.Status != frozenStatus {
		t.Fatalf("status = %q, want %q", file.Status, frozenStatus)
	}
	if len(file.RawCases) == 0 || len(file.ValueCases) == 0 {
		t.Fatal("legacy vector file must declare raw_cases and value_cases")
	}

	seenIDs := map[string]bool{}

	for _, tc := range file.RawCases {
		tc := tc
		if seenIDs[tc.ID] {
			t.Fatalf("duplicate case id %q", tc.ID)
		}
		seenIDs[tc.ID] = true

		t.Run(tc.ID, func(t *testing.T) {
			input := mustHex(t, tc.InputUTF8Hex)
			got, err := canonicalize.MarshalRaw(input)
			if tc.GoError != "" {
				if err == nil {
					t.Fatalf("expected Go error %q, got canonical output %s", tc.GoError, hex.EncodeToString(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("legacy MarshalRaw: %v", err)
			}
			if gotHex := hex.EncodeToString(got); gotHex != tc.GoCanonicalUTF8Hex {
				t.Fatalf("legacy canonical = %s, want %s", gotHex, tc.GoCanonicalUTF8Hex)
			}
		})
	}

	for _, tc := range file.ValueCases {
		tc := tc
		if seenIDs[tc.ID] {
			t.Fatalf("duplicate case id %q", tc.ID)
		}
		seenIDs[tc.ID] = true

		t.Run(tc.ID, func(t *testing.T) {
			var value any
			if err := json.Unmarshal([]byte(tc.ValueJSON), &value); err != nil {
				t.Fatalf("decode value_json: %v", err)
			}
			got, err := canonicalize.Marshal(value)
			if err != nil {
				t.Fatalf("legacy Marshal: %v", err)
			}
			if gotHex := hex.EncodeToString(got); gotHex != tc.Expected.Go {
				t.Fatalf("legacy canonical = %s, want %s", gotHex, tc.Expected.Go)
			}
			if tc.EqualAcrossLanguages {
				if tc.Expected.Go != tc.Expected.Python || tc.Expected.Go != tc.Expected.TypeScript {
					t.Fatalf(
						"equal_across_languages is true but expected values differ: go=%s python=%s typescript=%s",
						tc.Expected.Go,
						tc.Expected.Python,
						tc.Expected.TypeScript,
					)
				}
			}
		})
	}
}

func TestProfileSelectionCases(t *testing.T) {
	file := loadJSON[profileFile](t, profileCasesFile)
	if file.Schema != selectionSchema {
		t.Fatalf("schema = %q, want %q", file.Schema, selectionSchema)
	}
	if file.Status != selectionStatus {
		t.Fatalf("status = %q, want %q", file.Status, selectionStatus)
	}
	if file.Profiles.Legacy != legacyProfileID {
		t.Fatalf("profiles.legacy = %q, want %q", file.Profiles.Legacy, legacyProfileID)
	}
	if file.Profiles.Strict != strictProfileID {
		t.Fatalf("profiles.strict = %q, want %q", file.Profiles.Strict, strictProfileID)
	}
	if file.ManifestField != "canonical_profile" {
		t.Fatalf("manifest_field = %q, want canonical_profile", file.ManifestField)
	}
	if file.ManifestVersionStrict != 4 {
		t.Fatalf("manifest_version_strict = %d, want 4", file.ManifestVersionStrict)
	}
	if len(file.Cases) == 0 {
		t.Fatal("profile selection file has no cases")
	}

	seenIDs := map[string]bool{}
	for _, tc := range file.Cases {
		if seenIDs[tc.ID] {
			t.Fatalf("duplicate case id %q", tc.ID)
		}
		seenIDs[tc.ID] = true
		if tc.CurrentReader == "" {
			t.Fatalf("case %q is missing current_reader", tc.ID)
		}
		if tc.StrictReader == "" {
			t.Fatalf("case %q is missing strict_reader", tc.ID)
		}
	}
}

func TestFixtureCaseIDsAreUniqueAcrossFiles(t *testing.T) {
	seen := map[string]string{}

	add := func(file string, ids []string) {
		for _, id := range ids {
			if previous, ok := seen[id]; ok {
				t.Fatalf("case id %q appears in both %s and %s", id, previous, file)
			}
			seen[id] = file
		}
	}

	positive := loadJSON[positiveFile](t, positiveVectorsFile)
	positiveIDs := make([]string, 0, len(positive.Cases))
	for _, tc := range positive.Cases {
		positiveIDs = append(positiveIDs, tc.ID)
	}
	add(positiveVectorsFile, positiveIDs)

	negative := loadJSON[negativeFile](t, negativeVectorsFile)
	negativeIDs := make([]string, 0, len(negative.Cases))
	for _, tc := range negative.Cases {
		negativeIDs = append(negativeIDs, tc.ID)
	}
	add(negativeVectorsFile, negativeIDs)

	legacy := loadJSON[legacyFile](t, legacyVectorsFile)
	legacyIDs := make([]string, 0, len(legacy.RawCases)+len(legacy.ValueCases))
	for _, tc := range legacy.RawCases {
		legacyIDs = append(legacyIDs, tc.ID)
	}
	for _, tc := range legacy.ValueCases {
		legacyIDs = append(legacyIDs, tc.ID)
	}
	add(legacyVectorsFile, legacyIDs)

	selection := loadJSON[profileFile](t, profileCasesFile)
	selectionIDs := make([]string, 0, len(selection.Cases))
	for _, tc := range selection.Cases {
		selectionIDs = append(selectionIDs, tc.ID)
	}
	add(profileCasesFile, selectionIDs)
}

func ternary(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}
