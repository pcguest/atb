// SPDX-License-Identifier: MIT
package identity_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/identity"
)

func TestFileResolver(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "identity-map.yaml")
	content := "sk-test:\n  display_name: Test Operator\n  email: operator@example.com\n  org_role: admin\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write mapping: %v", err)
	}

	resolver := identity.FileResolver{Path: path}
	id, err := resolver.Resolve("sk-test")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id.DisplayName != "Test Operator" || id.Email != "operator@example.com" {
		t.Fatalf("identity = %#v", id)
	}
}

func TestEnvResolver(t *testing.T) {
	key := "sk-env"
	t.Setenv(identity.EnvVarName(key), "Env User|env@example.com|operator")
	id, err := (identity.EnvResolver{}).Resolve(key)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id.DisplayName != "Env User" || id.Email != "env@example.com" {
		t.Fatalf("identity = %#v", id)
	}
}

func TestChainResolverPrefersFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity-map.yaml")
	if err := os.WriteFile(path, []byte("sk-chain:\n  name: File User\n"), 0o600); err != nil {
		t.Fatalf("write mapping: %v", err)
	}
	chain := identity.ChainResolver{
		Resolvers: []identity.Resolver{
			identity.FileResolver{Path: path},
			identity.EnvResolver{},
		},
	}
	id, err := chain.Resolve("sk-chain")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id.DisplayName != "File User" {
		t.Fatalf("display_name = %q", id.DisplayName)
	}
}

func TestFallbackDisplayName(t *testing.T) {
	if got := identity.FallbackDisplayName("sk-abcd1234"); got != "api-key:1234" {
		t.Fatalf("fallback = %q", got)
	}
}

func TestWriteMapping(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity-map.yaml")
	if err := identity.WriteMapping(path, "sk-write", "Test Operator", "operator@example.com", "admin"); err != nil {
		t.Fatalf("WriteMapping: %v", err)
	}
	id, err := identity.FileResolver{Path: path}.Resolve("sk-write")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id.DisplayName != "Test Operator" {
		t.Fatalf("display_name = %q", id.DisplayName)
	}
}

func TestFileResolverErrorsAndFallbackName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		key     string
		wantErr error
		want    string
	}{
		{name: "empty key", key: "", wantErr: identity.ErrNotFound},
		{name: "missing file", key: "key", wantErr: identity.ErrNotFound},
		{name: "malformed yaml", content: "key: [", key: "key"},
		{name: "missing entry", content: "other:\n  name: Other\n", key: "key", wantErr: identity.ErrNotFound},
		{name: "missing display name", content: "key:\n  email: user@example.com\n", key: "key", wantErr: identity.ErrInvalidMapping},
		{name: "legacy name", content: "key:\n  name: Legacy Name\n", key: "key", want: "Legacy Name"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".yaml")
			if tc.content != "" {
				if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
					t.Fatalf("write mapping: %v", err)
				}
			}
			got, err := (identity.FileResolver{Path: path}).Resolve(tc.key)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if tc.name == "malformed yaml" {
				if err == nil {
					t.Fatal("expected malformed YAML error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got.DisplayName != tc.want {
				t.Fatalf("display name = %q, want %q", got.DisplayName, tc.want)
			}
		})
	}
}

func TestEnvResolverBoundaries(t *testing.T) {
	if _, err := (identity.EnvResolver{}).Resolve(""); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("empty key error = %v", err)
	}
	if _, err := (identity.EnvResolver{}).Resolve("unset"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("unset key error = %v", err)
	}

	key := "partial"
	t.Setenv(identity.EnvVarName(key), "Name Only")
	got, err := (identity.EnvResolver{}).Resolve(key)
	if err != nil {
		t.Fatalf("partial env mapping: %v", err)
	}
	if got.DisplayName != "Name Only" || got.Email != "" || got.OrgRole != "" {
		t.Fatalf("partial env identity = %+v", got)
	}

	blank := "blank"
	t.Setenv(identity.EnvVarName(blank), " |email@example.com|admin")
	if _, err := (identity.EnvResolver{}).Resolve(blank); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("blank display error = %v", err)
	}
}

type resolverFunc func(string) (identity.Identity, error)

func (fn resolverFunc) Resolve(key string) (identity.Identity, error) {
	return fn(key)
}

func TestChainResolverBoundaries(t *testing.T) {
	wantErr := errors.New("resolver failed")
	chain := identity.ChainResolver{Resolvers: []identity.Resolver{
		nil,
		resolverFunc(func(string) (identity.Identity, error) {
			return identity.Identity{}, wantErr
		}),
		resolverFunc(func(string) (identity.Identity, error) {
			return identity.Identity{}, identity.ErrNotFound
		}),
	}}
	if _, err := chain.Resolve("key"); !errors.Is(err, wantErr) {
		t.Fatalf("chain error = %v, want %v", err, wantErr)
	}
	if _, err := (identity.ChainResolver{}).Resolve("key"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("empty chain error = %v", err)
	}

	success := identity.ChainResolver{Resolvers: []identity.Resolver{
		resolverFunc(func(string) (identity.Identity, error) {
			return identity.Identity{DisplayName: "Resolved"}, nil
		}),
	}}
	got, err := success.Resolve("key")
	if err != nil || got.DisplayName != "Resolved" {
		t.Fatalf("successful chain = %+v, %v", got, err)
	}
}

func TestWriteMappingValidationUpdateAndActorApplication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "identity-map.yaml")
	if err := identity.WriteMapping(path, "", "Name", "", ""); !errors.Is(err, identity.ErrInvalidMapping) {
		t.Fatalf("empty key error = %v", err)
	}
	if err := identity.WriteMapping(path, "key", "", "", ""); !errors.Is(err, identity.ErrInvalidMapping) {
		t.Fatalf("empty name error = %v", err)
	}
	if err := identity.WriteMapping(path, "key", "First", "", ""); err != nil {
		t.Fatalf("initial mapping: %v", err)
	}
	if err := identity.WriteMapping(path, "key", "Updated", "updated@example.com", "admin"); err != nil {
		t.Fatalf("updated mapping: %v", err)
	}
	got, err := (identity.FileResolver{Path: path}).Resolve("key")
	if err != nil || got.DisplayName != "Updated" || got.OrgRole != "admin" {
		t.Fatalf("updated identity = %+v, %v", got, err)
	}

	identity.ApplyActor(nil, identity.Identity{}, "")
	data := map[string]any{}
	identity.ApplyActor(data, identity.Identity{}, "short")
	actor := data["actor"].(map[string]string)
	if actor["display_name"] != "api-key:hort" {
		t.Fatalf("fallback actor = %+v", actor)
	}
	identity.ApplyActor(data, identity.Identity{
		DisplayName: "Patrick",
		Email:       "patrick@example.com",
		OrgRole:     "admin",
	}, "")
	actor = data["actor"].(map[string]string)
	if actor["display_name"] != "Patrick" || actor["email"] == "" || actor["org_role"] != "admin" {
		t.Fatalf("resolved actor = %+v", actor)
	}

	if got := identity.FallbackDisplayName("abc"); got != "api-key:abc" {
		t.Fatalf("short fallback = %q", got)
	}
	if got := identity.FallbackDisplayName(""); got != "" {
		t.Fatalf("empty fallback = %q", got)
	}
}
