// SPDX-License-Identifier: MIT
package locator

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// parityCase mirrors one case in test/locator/parity_cases.json. The fixture is
// shared so ATB and downstream consumers can assert the same grammar
// acceptances and rejections.
type parityCase struct {
	Name       string `json:"name"`
	Input      string `json:"input"`
	OK         bool   `json:"ok"`
	Canonical  string `json:"canonical,omitempty"`
	ErrorClass string `json:"error_class,omitempty"`
	Seq        *int64 `json:"seq,omitempty"`
}

func loadParityCases(t *testing.T) []parityCase {
	t.Helper()
	data, err := os.ReadFile("../../test/locator/parity_cases.json")
	if err != nil {
		t.Fatalf("read parity fixtures: %v", err)
	}
	var cases []parityCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("decode parity fixtures: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("parity fixtures are empty")
	}
	return cases
}

func TestParityCases(t *testing.T) {
	for _, tc := range loadParityCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			loc, err := Parse(tc.Input)
			if !tc.OK {
				if err == nil {
					t.Fatalf("Parse(%q) succeeded, want %s error", tc.Input, tc.ErrorClass)
				}
				switch tc.ErrorClass {
				case "malformed":
					if !errors.Is(err, ErrMalformed) {
						t.Fatalf("Parse(%q) error = %v, want ErrMalformed", tc.Input, err)
					}
				case "unsupported_version":
					if !errors.Is(err, ErrUnsupportedVersion) {
						t.Fatalf("Parse(%q) error = %v, want ErrUnsupportedVersion", tc.Input, err)
					}
				default:
					t.Fatalf("case %q has unknown error_class %q", tc.Name, tc.ErrorClass)
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.Input, err)
			}
			if tc.Seq != nil && loc.EventSequence != *tc.Seq {
				t.Fatalf("Parse(%q) seq = %d, want %d", tc.Input, loc.EventSequence, *tc.Seq)
			}
			if tc.Canonical != "" && loc.String() != tc.Canonical {
				t.Fatalf("Parse(%q).String() = %q, want %q", tc.Input, loc.String(), tc.Canonical)
			}
			if tc.Canonical != "" {
				got, err := Format(loc.BundleHeadHash, loc.EventSequence, loc.RecordHash)
				if err != nil {
					t.Fatalf("Format(%q): %v", tc.Name, err)
				}
				if got != tc.Canonical {
					t.Fatalf("Format(%q) = %q, want %q", tc.Name, got, tc.Canonical)
				}
			}
		})
	}
}

func TestFormat(t *testing.T) {
	const head = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const record = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	got, err := Format(head, 4, "")
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := "atb://evidence/1/" + head + "?seq=4"
	if got != want {
		t.Fatalf("Format = %q, want %q", got, want)
	}

	got, err = Format(head, 0, record)
	if err != nil {
		t.Fatalf("Format with record: %v", err)
	}
	want = "atb://evidence/1/" + head + "?seq=0&record=" + record
	if got != want {
		t.Fatalf("Format = %q, want %q", got, want)
	}

	if _, err := Format(head, -1, ""); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Format(negative seq) = %v, want ErrMalformed", err)
	}
	if _, err := Format("nothex", 1, ""); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Format(bad head) = %v, want ErrMalformed", err)
	}
}
