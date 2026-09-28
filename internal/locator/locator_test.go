// SPDX-License-Identifier: MIT
package locator

import (
	"errors"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

const testHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testRecord = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestParseValid(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Locator
	}{
		{
			name: "head and seq",
			in:   "atb://evidence/1/" + testHead + "?seq=3",
			want: Locator{BundleHeadHash: testHead, EventSequence: 3},
		},
		{
			name: "head seq and record",
			in:   "atb://evidence/1/" + testHead + "?seq=1&record=" + testRecord,
			want: Locator{BundleHeadHash: testHead, EventSequence: 1, RecordHash: testRecord},
		},
		{
			name: "zero sequence (manifest)",
			in:   "atb://evidence/1/" + testHead + "?seq=0",
			want: Locator{BundleHeadHash: testHead, EventSequence: 0},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("Parse(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
			// Canonical round-trip: Parse(String(Parse(x))) == Parse(x).
			again, err := Parse(got.String())
			if err != nil {
				t.Fatalf("Parse(canonical %q): %v", got.String(), err)
			}
			if again != got {
				t.Fatalf("round-trip mismatch: %+v != %+v", again, got)
			}
		})
	}
}

func TestParseCanonicalisesParameterOrder(t *testing.T) {
	// Non-canonical order is accepted but String() emits canonical order.
	got, err := Parse("atb://evidence/1/" + testHead + "?record=" + testRecord + "&seq=2")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := "atb://evidence/1/" + testHead + "?seq=2&record=" + testRecord
	if got.String() != want {
		t.Fatalf("String() = %q, want %q", got.String(), want)
	}
}

func TestParseRejects(t *testing.T) {
	head := testHead
	cases := []struct {
		name string
		in   string
		want error
	}{
		{"empty", "", ErrMalformed},
		{"missing seq", "atb://evidence/1/" + head, ErrMalformed},
		{"empty seq", "atb://evidence/1/" + head + "?seq=", ErrMalformed},
		{"negative seq", "atb://evidence/1/" + head + "?seq=-1", ErrMalformed},
		{"leading zero", "atb://evidence/1/" + head + "?seq=01", ErrMalformed},
		{"duplicate seq", "atb://evidence/1/" + head + "?seq=1&seq=2", ErrMalformed},
		{"unknown param", "atb://evidence/1/" + head + "?seq=1&path=/etc/passwd", ErrMalformed},
		{"duplicate record", "atb://evidence/1/" + head + "?seq=1&record=" + testRecord + "&record=" + testRecord, ErrMalformed},
		{"uppercase head", "atb://evidence/1/" + strings.ToUpper(head) + "?seq=1", ErrMalformed},
		{"short head", "atb://evidence/1/abcd?seq=1", ErrMalformed},
		{"wrong scheme", "http://evidence/1/" + head + "?seq=1", ErrMalformed},
		{"uppercase scheme", "ATB://evidence/1/" + head + "?seq=1", ErrMalformed},
		{"uppercase authority", "atb://EVIDENCE/1/" + head + "?seq=1", ErrMalformed},
		{"wrong authority", "atb://other/1/" + head + "?seq=1", ErrMalformed},
		{"fragment", "atb://evidence/1/" + head + "?seq=1#x", ErrMalformed},
		{"empty fragment delimiter", "atb://evidence/1/" + head + "?seq=1#", ErrMalformed},
		{"trailing ampersand", "atb://evidence/1/" + head + "?seq=1&", ErrMalformed},
		{"doubled ampersand", "atb://evidence/1/" + head + "?seq=1&&record=" + testRecord, ErrMalformed},
		{"leading ampersand", "atb://evidence/1/" + head + "?&seq=1", ErrMalformed},
		{"non-canonical version", "atb://evidence/01/" + head + "?seq=1", ErrMalformed},
		{"percent encoding", "atb://evidence/1/" + head + "?seq=1&record=%20", ErrMalformed},
		{"bad version text", "atb://evidence/x/" + head + "?seq=1", ErrMalformed},
		{"extra path segment", "atb://evidence/1/" + head + "/x?seq=1", ErrMalformed},
		{"seq overflow", "atb://evidence/1/" + head + "?seq=99999999999999999999999", ErrMalformed},
		{"bad record", "atb://evidence/1/" + head + "?seq=1&record=nothex", ErrMalformed},
		{"empty record", "atb://evidence/1/" + head + "?seq=1&record=", ErrMalformed},
		{"unsupported version", "atb://evidence/2/" + head + "?seq=1", ErrUnsupportedVersion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.in)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded, want error", tc.in)
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("Parse(%q) error = %v, want errors.Is %v", tc.in, err, tc.want)
			}
		})
	}
}

func TestValidateRejectsNegativeSequence(t *testing.T) {
	loc := Locator{BundleHeadHash: testHead, EventSequence: -1}
	if err := loc.Validate(); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Validate(-1) = %v, want ErrMalformed", err)
	}
}

// buildBundle creates a manifest-bearing bundle with the given number of
// additional events and returns it.
func buildBundle(t *testing.T, extra int) *bundle.Bundle {
	t.Helper()
	b, err := bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV2})
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	for i := 0; i < extra; i++ {
		if err := b.Append("test.event", map[string]any{"i": i}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	return b
}

func headHash(b *bundle.Bundle) string {
	return b.Records[len(b.Records)-1].Hash
}

func TestResolve(t *testing.T) {
	b := buildBundle(t, 3)
	head := headHash(b)
	// Manifest is seq 0; appended events are seq 1..3.
	target := b.Records[2]

	t.Run("correct head and seq", func(t *testing.T) {
		loc := Locator{BundleHeadHash: head, EventSequence: target.Event.Sequence}
		res, err := Resolve(loc, b)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if res.Seq != target.Event.Sequence || res.Index != 2 {
			t.Fatalf("resolution = %+v, want seq %d index 2", res, target.Event.Sequence)
		}
		if res.RecordHashMatched {
			t.Fatalf("RecordHashMatched = true with no record hash supplied")
		}
	})

	t.Run("correct optional record hash", func(t *testing.T) {
		loc := Locator{BundleHeadHash: head, EventSequence: target.Event.Sequence, RecordHash: target.Hash}
		res, err := Resolve(loc, b)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if !res.RecordHashMatched {
			t.Fatalf("RecordHashMatched = false, want true")
		}
	})

	t.Run("wrong bundle head", func(t *testing.T) {
		loc := Locator{BundleHeadHash: strings.Repeat("c", 64), EventSequence: target.Event.Sequence}
		_, err := Resolve(loc, b)
		if !errors.Is(err, ErrBundleUnavailable) {
			t.Fatalf("Resolve = %v, want ErrBundleUnavailable", err)
		}
	})

	t.Run("out of range sequence", func(t *testing.T) {
		loc := Locator{BundleHeadHash: head, EventSequence: 9999}
		_, err := Resolve(loc, b)
		if !errors.Is(err, ErrEventNotFound) {
			t.Fatalf("Resolve = %v, want ErrEventNotFound", err)
		}
	})

	t.Run("incorrect optional record hash", func(t *testing.T) {
		loc := Locator{BundleHeadHash: head, EventSequence: target.Event.Sequence, RecordHash: strings.Repeat("d", 64)}
		_, err := Resolve(loc, b)
		if !errors.Is(err, ErrRecordHashMismatch) {
			t.Fatalf("Resolve = %v, want ErrRecordHashMismatch", err)
		}
	})

	t.Run("nil bundle", func(t *testing.T) {
		loc := Locator{BundleHeadHash: head, EventSequence: 1}
		_, err := Resolve(loc, nil)
		if !errors.Is(err, ErrBundleUnavailable) {
			t.Fatalf("Resolve(nil) = %v, want ErrBundleUnavailable", err)
		}
	})

	t.Run("empty bundle", func(t *testing.T) {
		loc := Locator{BundleHeadHash: head, EventSequence: 1}
		_, err := Resolve(loc, &bundle.Bundle{})
		if !errors.Is(err, ErrBundleUnavailable) {
			t.Fatalf("Resolve(empty) = %v, want ErrBundleUnavailable", err)
		}
	})

	t.Run("duplicate sequence resolves to the first match", func(t *testing.T) {
		// Simulate corruption: two records share a seq. The locator does not
		// enforce sequence uniqueness (that is an integrity property); it
		// resolves deterministically to the first match.
		dup := buildBundle(t, 3)
		dup.Records[2].Event.Sequence = dup.Records[1].Event.Sequence
		loc := Locator{BundleHeadHash: dup.Records[len(dup.Records)-1].Hash, EventSequence: dup.Records[1].Event.Sequence}
		res, err := Resolve(loc, dup)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if res.Index != 1 {
			t.Fatalf("resolved index = %d, want first match at 1", res.Index)
		}
	})
}
