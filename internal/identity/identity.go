// SPDX-License-Identifier: MIT
// Package identity resolves API keys to human-readable actor metadata.
package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Identity holds resolved actor metadata for capture events.
type Identity struct {
	DisplayName string
	Email       string
	OrgRole     string
}

// Resolver maps API keys to identities.
type Resolver interface {
	Resolve(apiKey string) (Identity, error)
}

// FileResolver loads identity mappings from a YAML file.
type FileResolver struct {
	Path string
}

// EnvResolver reads ATB_IDENTITY_<HASHED_KEY> environment variables.
type EnvResolver struct{}

// ChainResolver tries resolvers in order until one succeeds.
type ChainResolver struct {
	Resolvers []Resolver
}

var (
	// ErrNotFound indicates no identity mapping exists for the API key.
	ErrNotFound = errors.New("identity: not found")
	// ErrInvalidMapping indicates a mapping file entry is malformed.
	ErrInvalidMapping = errors.New("identity: invalid mapping")
)

var _ Resolver = FileResolver{}
var _ Resolver = EnvResolver{}
var _ Resolver = ChainResolver{}

// DefaultMapPath returns ~/.atb/identity-map.yaml.
func DefaultMapPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("identity: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".atb", "identity-map.yaml"), nil
}

// Resolve implements Resolver.
func (r FileResolver) Resolve(apiKey string) (Identity, error) {
	if strings.TrimSpace(apiKey) == "" {
		return Identity{}, ErrNotFound
	}
	path := r.Path
	if path == "" {
		var err error
		path, err = DefaultMapPath()
		if err != nil {
			return Identity{}, err
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Identity{}, ErrNotFound
		}
		return Identity{}, err
	}
	var doc map[string]fileEntry
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return Identity{}, fmt.Errorf("identity: parse mapping file: %w", err)
	}
	entry, ok := doc[apiKey]
	if !ok {
		return Identity{}, ErrNotFound
	}
	id, err := entry.identity()
	if err != nil {
		return Identity{}, err
	}
	return id, nil
}

type fileEntry struct {
	DisplayName string `yaml:"display_name"`
	Name        string `yaml:"name"`
	Email       string `yaml:"email"`
	OrgRole     string `yaml:"org_role"`
}

func (e fileEntry) identity() (Identity, error) {
	name := strings.TrimSpace(e.DisplayName)
	if name == "" {
		name = strings.TrimSpace(e.Name)
	}
	if name == "" {
		return Identity{}, fmt.Errorf("%w: missing display_name", ErrInvalidMapping)
	}
	return Identity{
		DisplayName: name,
		Email:       strings.TrimSpace(e.Email),
		OrgRole:     strings.TrimSpace(e.OrgRole),
	}, nil
}

// Resolve implements Resolver.
func (EnvResolver) Resolve(apiKey string) (Identity, error) {
	if strings.TrimSpace(apiKey) == "" {
		return Identity{}, ErrNotFound
	}
	envKey := EnvVarName(apiKey)
	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" {
		return Identity{}, ErrNotFound
	}
	parts := strings.Split(value, "|")
	id := Identity{DisplayName: strings.TrimSpace(parts[0])}
	if len(parts) > 1 {
		id.Email = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		id.OrgRole = strings.TrimSpace(parts[2])
	}
	if id.DisplayName == "" {
		return Identity{}, ErrNotFound
	}
	return id, nil
}

// Resolve implements Resolver.
func (c ChainResolver) Resolve(apiKey string) (Identity, error) {
	var lastErr error
	for _, resolver := range c.Resolvers {
		if resolver == nil {
			continue
		}
		id, err := resolver.Resolve(apiKey)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, ErrNotFound) {
			lastErr = err
		}
	}
	if lastErr != nil {
		return Identity{}, lastErr
	}
	return Identity{}, ErrNotFound
}

// DefaultChain returns file-then-env resolution order.
func DefaultChain() ChainResolver {
	path, _ := DefaultMapPath()
	return ChainResolver{
		Resolvers: []Resolver{
			FileResolver{Path: path},
			EnvResolver{},
		},
	}
}

// WriteMapping creates or updates an identity mapping entry.
func WriteMapping(path, apiKey, displayName, email, orgRole string) error {
	apiKey = strings.TrimSpace(apiKey)
	displayName = strings.TrimSpace(displayName)
	if apiKey == "" || displayName == "" {
		return fmt.Errorf("%w: key and display name are required", ErrInvalidMapping)
	}
	doc := map[string]fileEntry{}
	if raw, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(raw, &doc)
	} else if !os.IsNotExist(err) {
		return err
	}
	doc[apiKey] = fileEntry{
		DisplayName: displayName,
		Email:       strings.TrimSpace(email),
		OrgRole:     strings.TrimSpace(orgRole),
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

// FallbackDisplayName returns api-key:<last-4> when no mapping exists.
func FallbackDisplayName(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if len(apiKey) >= 4 {
		return "api-key:" + apiKey[len(apiKey)-4:]
	}
	if apiKey != "" {
		return "api-key:" + apiKey
	}
	return ""
}

// EnvVarName returns the environment variable name for a hashed API key.
func EnvVarName(apiKey string) string {
	return "ATB_IDENTITY_" + hashKey(apiKey)
}

func hashKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return strings.ToUpper(hex.EncodeToString(sum[:8]))
}

// ApplyActor enriches an event data map with actor fields when resolved.
func ApplyActor(data map[string]any, id Identity, apiKey string) {
	if data == nil {
		return
	}
	actor, _ := data["actor"].(map[string]string)
	if actor == nil {
		actor = map[string]string{}
	}
	if id.DisplayName != "" {
		actor["display_name"] = id.DisplayName
	} else if apiKey != "" {
		actor["display_name"] = FallbackDisplayName(apiKey)
	}
	if id.Email != "" {
		actor["email"] = id.Email
	}
	if id.OrgRole != "" {
		actor["org_role"] = id.OrgRole
	}
	if len(actor) > 0 {
		data["actor"] = actor
	}
}
