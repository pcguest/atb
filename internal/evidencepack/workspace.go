// SPDX-License-Identifier: MIT
package evidencepack

import (
	"os"
	"path/filepath"
	"sort"
)

// DiscoverWorkspaceBundles returns bundle.atb paths under root/sessions/*/bundle.atb.
// A missing sessions directory yields an empty slice without error. Session
// directories without bundle.atb are skipped.
func DiscoverWorkspaceBundles(root string) ([]string, error) {
	sessionsDir := filepath.Join(root, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bundlePath := filepath.Join(sessionsDir, entry.Name(), "bundle.atb")
		info, err := os.Stat(bundlePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		paths = append(paths, bundlePath)
	}
	sort.Strings(paths)
	return paths, nil
}
