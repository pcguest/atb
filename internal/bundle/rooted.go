// SPDX-License-Identifier: MIT
package bundle

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// LoadRooted reads and verifies a bundle through the supplied directory capability.
// The path is relative to root; no pathname is reopened outside that capability.
func LoadRooted(root *os.Root, path string) (*Bundle, error) {
	f, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := LoadReader(f)
	if err != nil {
		return nil, err
	}
	if len(b.Records) == 0 {
		return nil, ErrNoManifest
	}
	if b.Records[0].Event.Type != ManifestEventType {
		return nil, ErrNotABundle
	}
	if err := b.Verify(); err != nil {
		return nil, err
	}
	return b, nil
}

// SaveRooted preserves Save's serialization, atomic replacement and advisory
// locking while resolving every filesystem operation through root.
func (b *Bundle) SaveRooted(ctx context.Context, root *os.Root, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	release, err := lockPathRooted(root, path)
	if err != nil {
		return err
	}
	defer release()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, record := range b.Records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
	return WriteAtomicRooted(root, path, buf.Bytes())
}

// WriteAtomicRooted replaces a file using only operations relative to root.
// This is also used for agent session metadata.
func WriteAtomicRooted(root *os.Root, path string, data []byte) error {
	if err := root.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	tmpPath := filepath.Join(filepath.Dir(path), ".atb-"+hex.EncodeToString(random[:])+".tmp")
	tmp, err := root.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, bundleFileMode)
	if err != nil {
		return err
	}
	defer root.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := root.Rename(tmpPath, path); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := root.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func lockPathRooted(root *os.Root, path string) (func(), error) {
	lockName := path + lockSuffix
	f, err := root.OpenFile(lockName, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockDescriptor(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	// Keep the sidecar inode stable: removing a lock while another opener holds
	// its inode permits a second independent lock to be created at the same name.
	return func() { _ = unlockDescriptor(f); _ = f.Close() }, nil
}

// RootedRelativePath converts accepted absolute-inside-root paths to relative
// names. This lexical conversion is not the security boundary: os.Root is.
func RootedRelativePath(dataDir, path string) (string, error) {
	root, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(path) {
		path, err = filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
	}
	if !filepath.IsLocal(path) || filepath.Clean(path) == "." {
		return "", fmt.Errorf("bundle path escapes agent data directory")
	}
	return filepath.Clean(path), nil
}
