// SPDX-License-Identifier: MIT
// Package retentionaudit records retention policy and enforcement operations
// in a separate project-local ATB bundle.
package retentionaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

const (
	DirName  = ".atb"
	FileName = "operations.atb"
)

// DefaultPath returns the project-local retention operations bundle path.
func DefaultPath() string {
	return filepath.Join(DirName, FileName)
}

// PathForBundle locates the operations bundle beside an explicitly selected
// evidence bundle. The conventional run.atb directory is treated as a project
// child; arbitrary bundle files use their containing directory.
func PathForBundle(bundlePath string) string {
	dir := filepath.Dir(filepath.Clean(bundlePath))
	if filepath.Base(dir) == bundle.BundleDir {
		dir = filepath.Dir(dir)
	}
	return filepath.Join(dir, DirName, FileName)
}

// Digest returns a stable SHA-256 digest for a JSON-serialisable value.
func Digest(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("retention audit: marshal digest input: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// Append adds one event to a verified operations bundle, creating it when
// absent. A corrupt existing audit bundle is never overwritten.
func Append(path, eventType string, data any, at time.Time) error {
	var (
		b   *bundle.Bundle
		err error
	)
	b, err = bundle.LoadVerified(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("retention audit: load %s: %w", path, err)
		}
		b, err = bundle.New()
		if err != nil {
			return fmt.Errorf("retention audit: create bundle: %w", err)
		}
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if err := b.AppendWithOptions(eventType, data, &bundle.AppendOptions{
		Timestamp: at.UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return fmt.Errorf("retention audit: append %s: %w", eventType, err)
	}
	if err := b.Save(path); err != nil {
		return fmt.Errorf("retention audit: save %s: %w", path, err)
	}
	return nil
}
