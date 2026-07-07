// SPDX-License-Identifier: MIT
package evidencepack

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

func TestPackPathsProfileFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	if _, err := os.Stat(filepath.Join(root, "rag_answer-pass.atb")); os.IsNotExist(err) {
		t.Skip("profile fixtures not generated (*.atb is gitignored); run make test-go or go run ./scripts/generate_profile_fixtures.go")
	}
	tests := []struct {
		name        string
		wantPass    bool
		wantRisk    bool
		wantProfile string
	}{
		{name: "rag_answer-pass.atb", wantPass: true, wantProfile: "atb.profile.rag_answer"},
		{name: "rag_answer-fail.atb", wantRisk: true, wantProfile: "atb.profile.rag_answer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pack, anyErrors, userError := PackPaths(
				context.Background(),
				[]string{filepath.Join(root, tc.name)},
			)
			if anyErrors || userError {
				t.Fatalf("PackPaths errors=%v userError=%v pack=%+v", anyErrors, userError, pack)
			}
			if len(pack.Bundles) != 1 {
				t.Fatalf("bundle count = %d, want 1", len(pack.Bundles))
			}
			got := pack.Bundles[0]
			if !got.IntegrityPass || got.ProfilePass != tc.wantPass || got.ProfileID != tc.wantProfile {
				t.Fatalf("summary = %+v", got)
			}
			if tc.wantPass && (got.CASGrade == "" || got.HeadHash == "") {
				t.Fatalf("passing summary lacks grade or head hash: %+v", got)
			}
			if tc.wantRisk && got.ResidualRisk == nil {
				t.Fatalf("failing summary lacks residual risk: %+v", got)
			}
		})
	}
}

func TestPackPathsClassifiesInputErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.atb")
	pack, anyErrors, userError := PackPaths(context.Background(), []string{"", missing})
	if !anyErrors || !userError {
		t.Fatalf("errors=%v userError=%v", anyErrors, userError)
	}
	if len(pack.Bundles) != 2 {
		t.Fatalf("bundle count = %d, want 2", len(pack.Bundles))
	}
	if !strings.Contains(pack.Bundles[0].Error, "path is required") {
		t.Fatalf("empty-path error = %q", pack.Bundles[0].Error)
	}
	if pack.Bundles[1].Error == "" {
		t.Fatal("missing-path error is empty")
	}
}

func TestManifestValidationAndUserPathClassification(t *testing.T) {
	if err := validateBundleManifest(nil); err == nil {
		t.Fatal("nil bundle unexpectedly valid")
	}
	if err := validateBundleManifest(&bundle.Bundle{}); err == nil {
		t.Fatal("empty bundle unexpectedly valid")
	}
	notManifest := &bundle.Bundle{Records: []bundle.Record{{Event: event.Event{Type: "ai.request.received"}}}}
	if err := validateBundleManifest(notManifest); err == nil {
		t.Fatal("non-manifest first record unexpectedly valid")
	}

	if isUserPathError(nil) {
		t.Fatal("nil error classified as user path error")
	}
	if !isUserPathError(os.ErrNotExist) {
		t.Fatal("os.ErrNotExist not classified as user path error")
	}
	if !isUserPathError(errors.New("bundle not found")) {
		t.Fatal("not-found message not classified as user path error")
	}
	if !isUserPathError(errors.New("no such file")) {
		t.Fatal("no-such-file message not classified as user path error")
	}
	if isUserPathError(errors.New("verification failed")) {
		t.Fatal("unrelated error classified as user path error")
	}
}
