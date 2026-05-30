package receipt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Added: ErrReceiptNotFound lets callers map missing receipts to a stable 404.
var ErrReceiptNotFound = errors.New("receipt not found")

// Receipt represents a record of a bundle ingested into custody.
type Receipt struct {
	ExportVersion string          `json:"export_version"`
	ReceiptID     string          `json:"receipt_id"`
	BundleHash    string          `json:"bundle_hash"`
	SubmittedAt   string          `json:"submitted_at"`
	ProfileID     string          `json:"profile_id,omitempty"`
	SubmitterRef  string          `json:"submitter_ref,omitempty"`
	VerifyReport  json.RawMessage `json:"verify_report"` // Raw JSON from verify.report.v1
}

// WORMStore defines the interface for Write-Once, Read-Many storage.
type WORMStore interface {
	// Changed: Store returns the immutable storage location while preserving the existing context-aware call path.
	Store(ctx context.Context, bundleBytes []byte, bundleHash string) (string, error)
	Retrieve(ctx context.Context, receiptID string) ([]byte, error)
}

// ReceiptStore defines the interface for storing and retrieving Receipts.
type ReceiptStore interface {
	// Added: Save provides the filesystem store name while keeping existing callers compatible.
	Save(ctx context.Context, receipt Receipt) error
	// Added: Get provides the filesystem store name while keeping existing callers compatible.
	Get(ctx context.Context, receiptID string) (Receipt, error)
	// Added: List returns receipts in stable submitted-at order for custody index views.
	List(ctx context.Context) ([]Receipt, error)
	StoreReceipt(ctx context.Context, receipt Receipt) error
	GetReceipt(ctx context.Context, receiptID string) (Receipt, error)
}

// InMemoryReceiptStore is a simple in-memory implementation of ReceiptStore for testing.
type InMemoryReceiptStore struct {
	mu       sync.Mutex
	receipts map[string]Receipt
}

// NewInMemoryReceiptStore creates a new InMemoryReceiptStore.
func NewInMemoryReceiptStore() *InMemoryReceiptStore {
	return &InMemoryReceiptStore{
		receipts: make(map[string]Receipt),
	}
}

// StoreReceipt stores a Receipt in memory.
func (s *InMemoryReceiptStore) StoreReceipt(ctx context.Context, receipt Receipt) error {
	// Added: StoreReceipt delegates to Save so in-memory and filesystem stores share semantics.
	return s.Save(ctx, receipt)
}

// Save stores a Receipt in memory.
func (s *InMemoryReceiptStore) Save(ctx context.Context, receipt Receipt) error {
	// Added: Context cancellation must be honoured before mutating the in-memory receipt index.
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts[receipt.ReceiptID] = receipt
	return nil
}

// GetReceipt retrieves a Receipt from memory.
func (s *InMemoryReceiptStore) GetReceipt(ctx context.Context, receiptID string) (Receipt, error) {
	// Added: GetReceipt delegates to Get so missing receipts return the typed sentinel.
	return s.Get(ctx, receiptID)
}

// Get retrieves a Receipt from memory.
func (s *InMemoryReceiptStore) Get(ctx context.Context, receiptID string) (Receipt, error) {
	// Added: Context cancellation must be honoured before reading the in-memory receipt index.
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, ok := s.receipts[receiptID]
	if !ok {
		return Receipt{}, fmt.Errorf("%w: %s", ErrReceiptNotFound, receiptID)
	}
	return receipt, nil
}

// List returns receipts sorted by SubmittedAt ascending.
func (s *InMemoryReceiptStore) List(ctx context.Context) ([]Receipt, error) {
	// Added: Context cancellation must be honoured before listing the in-memory receipt index.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Added: An explicit slice avoids nil JSON arrays on empty receipt stores.
	receipts := make([]Receipt, 0, len(s.receipts))
	for _, receipt := range s.receipts {
		receipts = append(receipts, receipt)
	}
	// Added: Sorting by SubmittedAt gives deterministic custody receipt listings.
	sort.Slice(receipts, func(i, j int) bool {
		return receipts[i].SubmittedAt < receipts[j].SubmittedAt
	})
	return receipts, nil
}
