package receipt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileSystemReceiptStore stores receipt JSON documents below BaseDir.
// RetentionPolicy defines how old receipts are cleaned up.
type RetentionPolicy struct {
	MaxAgeDays int // Maximum age in days for receipts. 0 means no age limit.
	MaxCount   int // Maximum number of receipts to keep. 0 means no count limit.
}

// FileSystemReceiptStore stores receipt JSON documents below BaseDir.
type FileSystemReceiptStore struct {
	BaseDir string
	Version string // New: Storage layout version
	Policy  RetentionPolicy
}

// NewFileSystemReceiptStore creates a filesystem-backed receipt store.
func NewFileSystemReceiptStore(baseDir string, policy RetentionPolicy) *FileSystemReceiptStore {
	return &FileSystemReceiptStore{BaseDir: baseDir, Version: "v1", Policy: policy}
}

// StoreReceipt stores a receipt using the compatibility method name.
func (s *FileSystemReceiptStore) StoreReceipt(ctx context.Context, receipt Receipt) error {
	// Added: StoreReceipt delegates to Save so old ingest callers share the filesystem write path.
	return s.Save(ctx, receipt)
}

// Save writes a receipt JSON file with a temp-fsync-rename protocol.
func (s *FileSystemReceiptStore) Save(ctx context.Context, receipt Receipt) error {
	// Added: Context cancellation prevents receipt persistence after request shutdown.
	if err := ctx.Err(); err != nil {
		return err
	}
	// Added: A receipt ID is required because it becomes the on-disk JSON filename.
	if strings.TrimSpace(receipt.ReceiptID) == "" {
		return errors.New("receipt store: receipt id required")
	}
	// Added: The store creates its private base directory lazily with owner-only permissions.
	receiptDir := filepath.Join(s.BaseDir, s.Version)
	if err := os.MkdirAll(receiptDir, 0o700); err != nil {
		return fmt.Errorf("receipt store: create base directory: %w", err)
	}
	// Added: The receipt path is validated before any file is opened.
	finalPath, err := s.receiptPath(receipt.ReceiptID)
	if err != nil {
		return err
	}
	// Added: JSON bytes preserve the receipt wire object without adding external dependencies.
	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("receipt store: marshal receipt: %w", err)
	}
	// Added: The temporary path follows the required tmp-to-rename persistence protocol.
	tmpPath := finalPath + ".tmp"
	// Added: Opening with truncate only affects the temporary receipt file before atomic rename.
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("receipt store: create temp file: %w", err)
	}
	// Added: cleanupTemp removes partial data if any write or rename step fails.
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("receipt store: write temp file: %w", err)
	}
	// Added: File fsync completes the byte-level durability requirement before rename.
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("receipt store: fsync temp file: %w", err)
	}
	// Added: Closing after fsync releases the descriptor before the atomic rename.
	if err := file.Close(); err != nil {
		return fmt.Errorf("receipt store: close temp file: %w", err)
	}
	// Added: POSIX rename makes the completed receipt visible atomically and permits re-index overwrite.
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("receipt store: rename temp file: %w", err)
	}
	cleanupTemp = false
	// Added: Directory fsync makes the rename durable before success is reported.
	if err := fsyncDir(receiptDir); err != nil {
		return fmt.Errorf("receipt store: fsync base directory: %w", err)
	}
	return nil
}

// GetReceipt retrieves a receipt using the compatibility method name.
func (s *FileSystemReceiptStore) GetReceipt(ctx context.Context, receiptID string) (Receipt, error) {
	// Added: GetReceipt delegates to Get so old callers receive typed missing-receipt errors.
	return s.Get(ctx, receiptID)
}

// Get retrieves one receipt JSON file by receipt ID.
func (s *FileSystemReceiptStore) Get(ctx context.Context, receiptID string) (Receipt, error) {
	// Added: Context cancellation prevents unnecessary filesystem reads after request shutdown.
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	// Added: The receipt path is validated before reading from the configured store root.
	path, err := s.receiptPath(receiptID)
	if err != nil {
		return Receipt{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Receipt{}, fmt.Errorf("%w: %s", ErrReceiptNotFound, receiptID)
		}
		return Receipt{}, fmt.Errorf("receipt store: read receipt: %w", err)
	}
	var receipt Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return Receipt{}, fmt.Errorf("receipt store: unmarshal receipt: %w", err)
	}
	return receipt, nil
}

// List returns all readable receipt JSON files sorted by SubmittedAt ascending.
func (s *FileSystemReceiptStore) List(ctx context.Context) ([]Receipt, error) {
	// Added: Context cancellation prevents unnecessary directory scans after request shutdown.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Added: Missing store directories produce an empty list for first-run deployments.
	if _, err := os.Stat(s.BaseDir); errors.Is(err, os.ErrNotExist) {
		return []Receipt{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("receipt store: stat base directory: %w", err)
	}
	// Added: Glob limits listing to receipt JSON files without walking unrelated artefacts.
	paths, err := filepath.Glob(filepath.Join(s.BaseDir, s.Version, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("receipt store: glob receipts: %w", err)
	}
	// Added: An explicit slice avoids nil JSON arrays on empty receipt stores.
	receipts := make([]Receipt, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("receipt store: skip unreadable receipt %s: %v", path, err)
			continue
		}
		var receipt Receipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			log.Printf("receipt store: skip malformed receipt %s: %v", path, err)
			continue
		}
		receipts = append(receipts, receipt)
	}
	// Added: Sorting by SubmittedAt fulfils the current receipt schema's issued-time ordering.
	sort.Slice(receipts, func(i, j int) bool {
		return receipts[i].SubmittedAt.Before(receipts[j].SubmittedAt)
	})
	return receipts, nil
}

// CleanUp applies the configured retention policy to the stored receipts.
func (s *FileSystemReceiptStore) CleanUp(ctx context.Context) error {
	if s.Policy.MaxAgeDays == 0 && s.Policy.MaxCount == 0 {
		return nil // No retention policy configured
	}

	receipts, err := s.List(ctx)
	if err != nil {
		return fmt.Errorf("receipt store: cleanup list receipts: %w", err)
	}

	// Apply MaxAgeDays policy
	if s.Policy.MaxAgeDays > 0 {
		cutoff := time.Now().Add(time.Duration(-s.Policy.MaxAgeDays) * 24 * time.Hour)
		for _, receipt := range receipts {
			if receipt.SubmittedAt.Before(cutoff) {
				if err := s.deleteReceipt(ctx, receipt.ReceiptID); err != nil {
					log.Printf("receipt store: cleanup failed to delete old receipt %s: %v", receipt.ReceiptID, err)
				}
			}
		}
		// Re-list after age-based deletion to ensure accurate count for MaxCount policy
		receipts, err = s.List(ctx)
		if err != nil {
			return fmt.Errorf("receipt store: cleanup re-list receipts after age cleanup: %w", err)
		}
	}

	// Apply MaxCount policy
	if s.Policy.MaxCount > 0 && len(receipts) > s.Policy.MaxCount {
		// Receipts are already sorted by SubmittedAt ascending from List()
		toDelete := len(receipts) - s.Policy.MaxCount
		for i := 0; i < toDelete; i++ {
			if err := s.deleteReceipt(ctx, receipts[i].ReceiptID); err != nil {
				log.Printf("receipt store: cleanup failed to delete excess receipt %s: %v", receipts[i].ReceiptID, err)
			}
		}
	}
	return nil
}

// deleteReceipt removes a receipt file from the store.
func (s *FileSystemReceiptStore) deleteReceipt(ctx context.Context, receiptID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.receiptPath(receiptID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("receipt store: remove receipt file %s: %w", path, err)
	}
	return nil
}

// receiptPath resolves a receipt ID to a JSON file within BaseDir.
func (s *FileSystemReceiptStore) receiptPath(receiptID string) (string, error) {
	if strings.TrimSpace(s.BaseDir) == "" {
		return "", errors.New("receipt store: base directory required")
	}
	if strings.TrimSpace(receiptID) == "" {
		return "", fmt.Errorf("%w: empty receipt id", ErrReceiptNotFound)
	}
	if strings.ContainsAny(receiptID, `/\`) || filepath.Clean(receiptID) != receiptID {
		return "", errors.New("receipt store: invalid receipt id")
	}
	return filepath.Join(s.BaseDir, s.Version, receiptID+".json"), nil
}
