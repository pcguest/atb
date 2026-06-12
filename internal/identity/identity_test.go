// SPDX-License-Identifier: MIT
package identity_test

import (
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
