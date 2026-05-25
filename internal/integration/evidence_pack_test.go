//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"testing"

	evidencepack "github.com/pcguest/atb/internal/evidencepack"
)

func TestEvidencePackProfileFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "bundles", "profiles")
	cases := []struct {
		file        string
		profileID   string
		integrity   bool
		profilePass bool
		expectCAS   bool
		expectRisk  bool
	}{
		{"rag_answer-pass.atb", profileIDRAGAnswer, true, true, true, false},
		{"rag_answer-fail.atb", profileIDRAGAnswer, true, false, false, true},
		{"privileged_tool_action-pass.atb", profileIDPrivilegedToolAction, true, true, true, false},
		{"privileged_tool_action-fail.atb", profileIDPrivilegedToolAction, true, false, false, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			path := filepath.Join(root, tc.file)
			pack, anyErrors, userError := evidencepack.PackPaths(context.Background(), []string{path})
			if anyErrors || userError {
				t.Fatalf("PackPaths() errors=%v userError=%v", anyErrors, userError)
			}
			if len(pack.Bundles) != 1 {
				t.Fatalf("bundles len = %d, want 1", len(pack.Bundles))
			}
			entry := pack.Bundles[0]
			if entry.Error != "" {
				t.Fatalf("unexpected error: %s", entry.Error)
			}
			if entry.ProfileID != tc.profileID {
				t.Fatalf("profile_id = %q, want %q", entry.ProfileID, tc.profileID)
			}
			if entry.IntegrityPass != tc.integrity {
				t.Fatalf("integrity_pass = %v, want %v", entry.IntegrityPass, tc.integrity)
			}
			if entry.ProfilePass != tc.profilePass {
				t.Fatalf("profile_pass = %v, want %v", entry.ProfilePass, tc.profilePass)
			}
			if tc.expectCAS && entry.CASGrade == "" {
				t.Fatal("expected cas_grade for passing fixture")
			}
			if tc.expectRisk && entry.ResidualRisk == nil {
				t.Fatal("expected residual_risk for failing fixture")
			}
		})
	}
}

func TestEvidencePackMissingBundleEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.atb")
	pack, anyErrors, userError := evidencepack.PackPaths(context.Background(), []string{path})
	if !anyErrors || !userError {
		t.Fatalf("PackPaths() anyErrors=%v userError=%v, want both true", anyErrors, userError)
	}
	if len(pack.Bundles) != 1 {
		t.Fatalf("bundles len = %d, want 1", len(pack.Bundles))
	}
	if pack.Bundles[0].Error == "" {
		t.Fatal("expected error field for missing bundle")
	}
}
