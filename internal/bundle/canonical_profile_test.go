// SPDX-License-Identifier: MIT

package bundle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pcguest/atb/v2/internal/event"
	"github.com/pcguest/atb/v2/internal/hash"
)

type profileSelectionProfiles struct {
	Legacy string `json:"legacy"`

	Strict string `json:"strict"`
}

type profileSelectionCase struct {
	ID string `json:"id"`

	ManifestVersion int `json:"manifest_version"`

	CanonicalProfile string `json:"canonical_profile"`

	CurrentReader string `json:"current_reader"`

	StrictReader string `json:"strict_reader"`
}

type profileSelectionFixture struct {
	Schema string `json:"schema"`

	Status string `json:"status"`

	Profiles profileSelectionProfiles `json:"profiles"`

	ManifestField string `json:"manifest_field"`

	ManifestVersionStrict int `json:"manifest_version_strict"`

	Cases []profileSelectionCase `json:"cases"`
}

func loadProfileSelectionFixture(t *testing.T) profileSelectionFixture {
	t.Helper()
	path := filepath.Join("..", "..", "test", "jcs", "profile_selection_cases.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read profile selection fixture: %v", err)
	}
	var fixture profileSelectionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode profile selection fixture: %v", err)
	}
	return fixture
}

// TestCurrentReaderProfileSelection pins the fail-closed behaviour of the
// current reader against the authored profile-selection cases. A v3 manifest
// carrying canonical_profile is still legacy; a v4 manifest is rejected by
// this reader regardless of the declared profile.
func TestCurrentReaderProfileSelection(t *testing.T) {
	fixture := loadProfileSelectionFixture(t)
	if fixture.Schema != "atb.jcs.profile-selection-cases.v1" {
		t.Fatalf("schema = %q", fixture.Schema)
	}
	if fixture.Profiles.Legacy != "atb-jcs-legacy-1" {
		t.Fatalf("legacy profile = %q", fixture.Profiles.Legacy)
	}
	if fixture.Profiles.Strict != "atb-jcs-rfc8785-1" {
		t.Fatalf("strict profile = %q", fixture.Profiles.Strict)
	}
	if fixture.ManifestField != "canonical_profile" {
		t.Fatalf("manifest_field = %q", fixture.ManifestField)
	}
	if fixture.ManifestVersionStrict != 4 {
		t.Fatalf("manifest_version_strict = %d", fixture.ManifestVersionStrict)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("profile selection fixture has no cases")
	}

	for _, tc := range fixture.Cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			data := map[string]any{
				"version":    tc.ManifestVersion,
				"bundle_id":  "b",
				"created_at": "2026-01-02T03:04:05Z",
			}
			if tc.CanonicalProfile != "" {
				data["canonical_profile"] = tc.CanonicalProfile
			}
			record := Record{Event: hash.Event{
				Type: event.TypeBundleManifest,
				Data: data,
			}}

			manifest, err := parseManifestData(&record)
			switch tc.CurrentReader {
			case "accepts-as-legacy":
				if err != nil {
					t.Fatalf("current reader rejected %s: %v", tc.ID, err)
				}
				if manifest.Version != strconv.Itoa(tc.ManifestVersion) {
					t.Fatalf("manifest version = %q, want %d", manifest.Version, tc.ManifestVersion)
				}
			case "rejects":
				if !errors.Is(err, ErrMalformed) {
					t.Fatalf("current reader: want ErrMalformed, got %v", err)
				}
			default:
				t.Fatalf("unknown current_reader %q", tc.CurrentReader)
			}
		})
	}
}

// TestManifestVersionMaxIsFrozenAtThree pins the reader's forward-compatibility
// ceiling. Adding strict-profile manifest v4 support requires a versioned
// change to this expectation, not a silent widening.
func TestManifestVersionMaxIsFrozenAtThree(t *testing.T) {
	if ManifestVersionMax != 3 {
		t.Fatalf("ManifestVersionMax = %d, want 3", ManifestVersionMax)
	}
}
