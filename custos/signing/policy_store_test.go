// SPDX-License-Identifier: MIT
package signing_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pcguest/custos/signing"
)

func validPolicy() signing.SigningPolicy {
	return signing.SigningPolicy{
		OrgID:            "org-acme",
		KeySource:        signing.KeySourceLocalFile,
		KeyRef:           "/keys/acme.ed25519",
		RotationSchedule: "0 0 * * 0",
		RequireTSA:       true,
		CreatedAt:        time.Unix(1_700_000_000, 0).UTC(),
	}
}

func TestSigningPolicy_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*signing.SigningPolicy)
		wantErr bool
	}{
		{"valid", func(*signing.SigningPolicy) {}, false},
		{"empty rotation schedule allowed", func(p *signing.SigningPolicy) { p.RotationSchedule = "" }, false},
		{"kms key source", func(p *signing.SigningPolicy) { p.KeySource = signing.KeySourceKMS }, false},
		{"missing org id", func(p *signing.SigningPolicy) { p.OrgID = "  " }, true},
		{"missing key ref", func(p *signing.SigningPolicy) { p.KeyRef = "" }, true},
		{"unknown key source", func(p *signing.SigningPolicy) { p.KeySource = signing.KeySource(7) }, true},
		{"too few cron fields", func(p *signing.SigningPolicy) { p.RotationSchedule = "0 0 * *" }, true},
		{"too many cron fields", func(p *signing.SigningPolicy) { p.RotationSchedule = "0 0 * * 0 0" }, true},
		{"garbage cron field", func(p *signing.SigningPolicy) { p.RotationSchedule = "0 0 * * mon" }, true},
		{"step cron accepted", func(p *signing.SigningPolicy) { p.RotationSchedule = "*/15 0 1-7 * *" }, false},
		{"list cron accepted", func(p *signing.SigningPolicy) { p.RotationSchedule = "0 0,12 * * 1,3,5" }, false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy := validPolicy()
			tc.mutate(&policy)
			err := policy.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("Validate() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestInMemoryPolicyStore_RoundTrip(t *testing.T) {
	t.Parallel()

	store := signing.NewInMemoryPolicyStore()
	policy := validPolicy()
	if err := store.Save(policy); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(policy.OrgID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.OrgID != policy.OrgID || got.KeyRef != policy.KeyRef || got.RequireTSA != policy.RequireTSA {
		t.Fatalf("Get returned %+v, want %+v", *got, policy)
	}
}

func TestInMemoryPolicyStore_GetMissing(t *testing.T) {
	t.Parallel()

	store := signing.NewInMemoryPolicyStore()
	_, err := store.Get("nobody")
	if !errors.Is(err, signing.ErrPolicyNotFound) {
		t.Fatalf("Get missing = %v, want ErrPolicyNotFound", err)
	}
}

func TestInMemoryPolicyStore_RejectsInvalid(t *testing.T) {
	t.Parallel()

	store := signing.NewInMemoryPolicyStore()
	bad := validPolicy()
	bad.KeyRef = ""
	if err := store.Save(bad); err == nil {
		t.Fatal("Save invalid = nil, want error")
	}
	if _, err := store.Get(bad.OrgID); !errors.Is(err, signing.ErrPolicyNotFound) {
		t.Fatalf("invalid policy was persisted: %v", err)
	}
}

func TestInMemoryPolicyStore_SaveReplaces(t *testing.T) {
	t.Parallel()

	store := signing.NewInMemoryPolicyStore()
	policy := validPolicy()
	if err := store.Save(policy); err != nil {
		t.Fatalf("Save: %v", err)
	}
	policy.RequireTSA = false
	policy.KeyRef = "/keys/acme-v2.ed25519"
	if err := store.Save(policy); err != nil {
		t.Fatalf("Save replace: %v", err)
	}
	got, err := store.Get(policy.OrgID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RequireTSA || got.KeyRef != "/keys/acme-v2.ed25519" {
		t.Fatalf("Save did not replace prior policy: %+v", *got)
	}
	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len = %d, want 1 (replace, not append)", len(list))
	}
}

func TestInMemoryPolicyStore_ListSorted(t *testing.T) {
	t.Parallel()

	store := signing.NewInMemoryPolicyStore()
	for _, id := range []string{"org-c", "org-a", "org-b"} {
		policy := validPolicy()
		policy.OrgID = id
		if err := store.Save(policy); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}
	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{"org-a", "org-b", "org-c"}
	if len(list) != len(want) {
		t.Fatalf("List len = %d, want %d", len(list), len(want))
	}
	for i, p := range list {
		if p.OrgID != want[i] {
			t.Fatalf("List[%d].OrgID = %q, want %q", i, p.OrgID, want[i])
		}
	}
}

func TestFileSystemPolicyStore_RoundTrip(t *testing.T) {
	t.Parallel()

	// Point the store at a subdirectory it must create itself, so the
	// owner-only permission assertion exercises the store's own MkdirAll.
	dir := filepath.Join(t.TempDir(), "policies")
	store := signing.NewFileSystemPolicyStore(dir)
	policy := validPolicy()
	if err := store.Save(policy); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(policy.OrgID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.OrgID != policy.OrgID || got.KeyRef != policy.KeyRef ||
		got.RequireTSA != policy.RequireTSA || got.RotationSchedule != policy.RotationSchedule {
		t.Fatalf("Get returned %+v, want %+v", *got, policy)
	}

	// The store directory must be owner-only because it references key material.
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("base dir perm = %o, want 700", perm)
	}
}

func TestFileSystemPolicyStore_GetMissing(t *testing.T) {
	t.Parallel()

	store := signing.NewFileSystemPolicyStore(t.TempDir())
	_, err := store.Get("nobody")
	if !errors.Is(err, signing.ErrPolicyNotFound) {
		t.Fatalf("Get missing = %v, want ErrPolicyNotFound", err)
	}
}

func TestFileSystemPolicyStore_RejectsInvalidOrgID(t *testing.T) {
	t.Parallel()

	store := signing.NewFileSystemPolicyStore(t.TempDir())
	policy := validPolicy()
	policy.OrgID = "../escape"
	if err := store.Save(policy); err == nil {
		t.Fatal("Save with traversal org id = nil, want error")
	}
	if _, err := store.Get("../escape"); err == nil {
		t.Fatal("Get with traversal org id = nil, want error")
	}
}

func TestFileSystemPolicyStore_RejectsInvalidPolicy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := signing.NewFileSystemPolicyStore(dir)
	bad := validPolicy()
	bad.RotationSchedule = "not a cron"
	if err := store.Save(bad); err == nil {
		t.Fatal("Save invalid policy = nil, want error")
	}
	// Nothing should have been written.
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			t.Fatalf("invalid policy left a file on disk: %s", e.Name())
		}
	}
}

func TestFileSystemPolicyStore_SaveReplaces(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := signing.NewFileSystemPolicyStore(dir)
	policy := validPolicy()
	if err := store.Save(policy); err != nil {
		t.Fatalf("Save: %v", err)
	}
	policy.RequireTSA = false
	if err := store.Save(policy); err != nil {
		t.Fatalf("Save replace: %v", err)
	}
	got, err := store.Get(policy.OrgID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RequireTSA {
		t.Fatal("Save did not replace prior policy")
	}
}
