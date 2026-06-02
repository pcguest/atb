// SPDX-License-Identifier: MIT
package signing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ErrPolicyNotFound is returned when no signing policy exists for an org ID.
var ErrPolicyNotFound = errors.New("signing: policy not found")

// cronField matches one whitespace-separated field of a 5-field cron schedule:
// a wildcard, a number, a range, a step, or a comma list of those. It is a
// structural check, not a full cron evaluator — enough to reject obvious
// garbage while accepting standard schedules like "0 0 * * 0".
var cronField = regexp.MustCompile(`^(\*|\d+)(-\d+)?(/\d+)?(,(\*|\d+)(-\d+)?(/\d+)?)*$`)

// Validate reports whether the policy is well-formed enough to persist. Custos
// records this configuration; it does not itself sign bundles, so validation is
// limited to the fields an operator must get right for ATB core to act on the
// policy later.
func (p SigningPolicy) Validate() error {
	if strings.TrimSpace(p.OrgID) == "" {
		return errors.New("signing: org id required")
	}
	switch p.KeySource {
	case KeySourceLocalFile, KeySourceKMS:
	default:
		return fmt.Errorf("signing: unknown key source %d", p.KeySource)
	}
	if strings.TrimSpace(p.KeyRef) == "" {
		return errors.New("signing: key reference required")
	}
	if err := validateRotationSchedule(p.RotationSchedule); err != nil {
		return err
	}
	return nil
}

// validateRotationSchedule accepts an empty schedule (rotation disabled) or a
// standard 5-field cron expression.
func validateRotationSchedule(schedule string) error {
	schedule = strings.TrimSpace(schedule)
	if schedule == "" {
		return nil
	}
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return fmt.Errorf("signing: rotation schedule must have 5 cron fields, got %d", len(fields))
	}
	for i, field := range fields {
		if !cronField.MatchString(field) {
			return fmt.Errorf("signing: rotation schedule field %d (%q) is not a valid cron field", i+1, field)
		}
	}
	return nil
}

// InMemoryPolicyStore is a thread-safe PolicyStore for tests and single-process
// use. Policies do not survive a restart.
type InMemoryPolicyStore struct {
	mu       sync.RWMutex
	policies map[string]SigningPolicy
}

// NewInMemoryPolicyStore creates an empty in-memory policy store.
func NewInMemoryPolicyStore() *InMemoryPolicyStore {
	return &InMemoryPolicyStore{policies: make(map[string]SigningPolicy)}
}

// Save validates and stores the policy, keyed by OrgID. Saving an existing
// OrgID replaces the prior policy.
func (s *InMemoryPolicyStore) Save(policy SigningPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[policy.OrgID] = policy
	return nil
}

// Get returns the policy for an org ID, or ErrPolicyNotFound.
func (s *InMemoryPolicyStore) Get(orgID string) (*SigningPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.policies[orgID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPolicyNotFound, orgID)
	}
	out := policy
	return &out, nil
}

// List returns all stored policies sorted by OrgID.
func (s *InMemoryPolicyStore) List() ([]SigningPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SigningPolicy, 0, len(s.policies))
	for _, policy := range s.policies {
		out = append(out, policy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OrgID < out[j].OrgID })
	return out, nil
}

var _ PolicyStore = (*InMemoryPolicyStore)(nil)

// FileSystemPolicyStore persists one JSON document per org below BaseDir. A
// signing policy references key material (a local key path or KMS key ID), so
// the store creates its directory and files owner-only.
type FileSystemPolicyStore struct {
	BaseDir string
}

// NewFileSystemPolicyStore creates a filesystem-backed policy store.
func NewFileSystemPolicyStore(baseDir string) *FileSystemPolicyStore {
	return &FileSystemPolicyStore{BaseDir: baseDir}
}

// Save validates the policy and writes it atomically via a temp-fsync-rename
// protocol. Saving an existing OrgID replaces the prior policy.
func (s *FileSystemPolicyStore) Save(policy SigningPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.BaseDir, 0o700); err != nil {
		return fmt.Errorf("signing: create base directory: %w", err)
	}
	finalPath, err := s.policyPath(policy.OrgID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("signing: marshal policy: %w", err)
	}
	tmpPath := finalPath + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("signing: create temp file: %w", err)
	}
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("signing: write temp file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("signing: fsync temp file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("signing: close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("signing: rename temp file: %w", err)
	}
	cleanupTemp = false
	return nil
}

// Get reads one policy JSON file by org ID, or returns ErrPolicyNotFound.
func (s *FileSystemPolicyStore) Get(orgID string) (*SigningPolicy, error) {
	path, err := s.policyPath(orgID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrPolicyNotFound, orgID)
		}
		return nil, fmt.Errorf("signing: read policy: %w", err)
	}
	var policy SigningPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("signing: unmarshal policy: %w", err)
	}
	return &policy, nil
}

// policyPath resolves an org ID to a JSON file within BaseDir, rejecting ids
// that would escape the store directory.
func (s *FileSystemPolicyStore) policyPath(orgID string) (string, error) {
	if strings.TrimSpace(s.BaseDir) == "" {
		return "", errors.New("signing: base directory required")
	}
	if strings.TrimSpace(orgID) == "" {
		return "", fmt.Errorf("%w: empty org id", ErrPolicyNotFound)
	}
	if strings.ContainsAny(orgID, `/\`) || filepath.Clean(orgID) != orgID {
		return "", errors.New("signing: invalid org id")
	}
	return filepath.Join(s.BaseDir, orgID+".json"), nil
}

var _ PolicyStore = (*FileSystemPolicyStore)(nil)
