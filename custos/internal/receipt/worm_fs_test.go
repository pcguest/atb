package receipt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestWORMStoreRetrieveRejectsTraversal(t *testing.T) {
	t.Parallel()
	store := NewFileSystemWORMStore(t.TempDir())
	ctx := context.Background()
	// Each of these would escape the configured WORM root if joined naively.
	for _, receiptID := range []string{
		"../escape",
		"sha256-../escape",
		"/etc/passwd",
		"sub/dir",
		"a/../../b",
		`..\windows`,
	} {
		if _, err := store.Retrieve(ctx, receiptID); err == nil {
			t.Errorf("Retrieve(%q) = nil error, want rejection", receiptID)
		}
	}
}

func TestWORMStoreRoundtrip(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory keeps filesystem persistence tests isolated.
	baseDir := t.TempDir()
	// Added: The filesystem store writes immutable bundle bytes below the configured base path.
	store := NewFileSystemWORMStore(baseDir)
	// Added: The sample bytes stand in for a verified ATB bundle.
	bundleBytes := []byte("verified atb bundle bytes")
	// Added: The head hash supplied to Store must match the bundle bytes.
	headHash := testHeadHash(bundleBytes)

	// Added: Store returns the immutable file path for the persisted bundle.
	path, err := store.Store(context.Background(), bundleBytes, headHash)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat stored bundle: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(bundleBytes) {
		t.Fatalf("stored bytes mismatch: got %q want %q", string(got), string(bundleBytes))
	}
}

func TestWORMStoreIdempotent(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory keeps idempotency assertions local to this test.
	baseDir := t.TempDir()
	// Added: The filesystem store must not overwrite an existing content-addressed bundle.
	store := NewFileSystemWORMStore(baseDir)
	// Added: The same bytes produce the same verified content hash.
	bundleBytes := []byte("same atb bundle bytes")
	// Added: The head hash is the idempotency key for the immutable file.
	headHash := testHeadHash(bundleBytes)

	firstPath, err := store.Store(context.Background(), bundleBytes, headHash)
	if err != nil {
		t.Fatalf("first Store: %v", err)
	}
	secondPath, err := store.Store(context.Background(), bundleBytes, headHash)
	if err != nil {
		t.Fatalf("second Store: %v", err)
	}
	if secondPath != firstPath {
		t.Fatalf("expected same path, got %q want %q", secondPath, firstPath)
	}
	files, err := filepath.Glob(filepath.Join(baseDir, "*.atb"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one stored bundle, got %d", len(files))
	}
}

func TestWORMStoreHashMismatch(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory proves mismatched writes leave no persisted artefact.
	baseDir := t.TempDir()
	// Added: The filesystem store rejects bytes that do not match the supplied head hash.
	store := NewFileSystemWORMStore(baseDir)
	// Added: The bundle bytes deliberately do not match the supplied hash.
	bundleBytes := []byte("mismatched atb bundle bytes")

	if _, err := store.Store(context.Background(), bundleBytes, "not-the-real-hash"); err == nil {
		t.Fatal("expected hash mismatch error")
	}
	files, err := filepath.Glob(filepath.Join(baseDir, "*.atb"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no stored bundles, got %d", len(files))
	}
}

func testHeadHash(bundleBytes []byte) string {
	// Added: Tests use the same SHA-256 hex form enforced by FileSystemWORMStore.
	hash := sha256.Sum256(bundleBytes)
	return hex.EncodeToString(hash[:])
}
