package receipt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemWORMStore stores immutable ATB bundles below BaseDir.
type FileSystemWORMStore struct {
	BaseDir string
}

// NewFileSystemWORMStore creates a filesystem-backed WORM store.
func NewFileSystemWORMStore(baseDir string) *FileSystemWORMStore {
	// Added: The constructor keeps daemon wiring compact while exposing only the configured base path.
	return &FileSystemWORMStore{BaseDir: baseDir}
}

// Store writes bundle bytes once under their SHA-256 content hash.
func (s *FileSystemWORMStore) Store(ctx context.Context, bundleBytes []byte, bundleHash string) (string, error) {
	// Added: Context cancellation prevents persistence after the caller has abandoned ingest.
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// The store is content-addressed: the supplied hash must be the SHA-256 of
	// the bytes being stored. This is a storage-boundary integrity check (the
	// bytes written match their key), not the bundle's hash-chain head hash.
	actualHash := sha256.Sum256(bundleBytes)
	actualHex := hex.EncodeToString(actualHash[:])
	if actualHex != bundleHash {
		return "", fmt.Errorf("worm store: content hash mismatch: got %s want %s", actualHex, bundleHash)
	}
	if strings.TrimSpace(s.BaseDir) == "" {
		return "", errors.New("worm store: base directory required")
	}
	// Added: The store creates its private base directory lazily with owner-only permissions.
	if err := os.MkdirAll(s.BaseDir, 0o700); err != nil {
		return "", fmt.Errorf("worm store: create base directory: %w", err)
	}

	// Added: Bundle files are named directly by the verified SHA-256 content hash.
	finalPath := filepath.Join(s.BaseDir, actualHex+".atb")
	if _, err := os.Stat(finalPath); err == nil {
		return finalPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("worm store: stat final path: %w", err)
	}

	// Added: The temporary path follows the required tmp-to-rename persistence protocol.
	tmpPath := finalPath + ".tmp"
	// Added: Opening with truncate only affects the temporary file, never an existing immutable bundle.
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("worm store: create temp file: %w", err)
	}
	// Added: cleanupTemp removes partial data if any write or rename step fails.
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := file.Write(bundleBytes); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("worm store: write temp file: %w", err)
	}
	// Added: File fsync completes the byte-level durability requirement before rename.
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("worm store: fsync temp file: %w", err)
	}
	// Added: Closing after fsync releases the descriptor before the atomic rename.
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("worm store: close temp file: %w", err)
	}
	if _, err := os.Stat(finalPath); err == nil {
		return finalPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("worm store: stat final path before rename: %w", err)
	}
	// Added: POSIX rename makes the completed temp file visible atomically.
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", fmt.Errorf("worm store: rename temp file: %w", err)
	}
	cleanupTemp = false
	// Added: Directory fsync makes the rename durable before success is reported.
	if err := fsyncDir(s.BaseDir); err != nil {
		return "", fmt.Errorf("worm store: fsync base directory: %w", err)
	}
	return finalPath, nil
}

// Retrieve reads immutable bundle bytes by receipt ID.
func (s *FileSystemWORMStore) Retrieve(ctx context.Context, receiptID string) ([]byte, error) {
	// Added: Context cancellation prevents unnecessary filesystem reads after request shutdown.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Added: Receipt IDs map to the verified bundle hash filename used at ingest.
	path, err := s.bundlePath(receiptID)
	if err != nil {
		return nil, err
	}
	bundleBytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrReceiptNotFound, receiptID)
		}
		return nil, fmt.Errorf("worm store: read bundle: %w", err)
	}
	return bundleBytes, nil
}

// bundlePath resolves a receipt ID to a bundle file below BaseDir.
func (s *FileSystemWORMStore) bundlePath(receiptID string) (string, error) {
	// Added: Retrieval must remain scoped to the configured WORM directory.
	if strings.TrimSpace(s.BaseDir) == "" {
		return "", errors.New("worm store: base directory required")
	}
	if strings.TrimSpace(receiptID) == "" {
		return "", fmt.Errorf("%w: empty receipt id", ErrReceiptNotFound)
	}
	// Changed: Absolute paths are rejected so receipt lookup cannot escape the WORM root.
	if filepath.IsAbs(receiptID) {
		return "", fmt.Errorf("worm store: invalid receipt id")
	}
	hash := strings.TrimPrefix(receiptID, "sha256-")
	if strings.ContainsAny(hash, `/\`) {
		return "", fmt.Errorf("worm store: invalid receipt id")
	}
	return filepath.Join(s.BaseDir, hash+".atb"), nil
}

// fsyncDir fsyncs a directory entry after a rename or overwrite.
func fsyncDir(path string) error {
	// Added: Opening the directory lets POSIX filesystems flush the rename metadata.
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
