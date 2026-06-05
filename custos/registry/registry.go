// SPDX-License-Identifier: MIT

// Package registry provides the Custos receipt + digest registry: an index over
// ingested receipts supporting lookup by receipt ID and by bundle hash (the
// reverse, digest-keyed lookup the receipt store does not provide). An auditor
// typically holds a bundle's hash, not its content-addressed receipt ID, so the
// digest index answers "which receipts custody this bundle?".
//
// The registry never mutates receipt content — receipts are immutable custody
// records. It only indexes them: Register is an idempotent upsert keyed by
// receipt ID, so re-ingesting the same receipt refreshes the index in place
// without duplicating entries.
package registry

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/pcguest/custos/internal/receipt"
)

// ErrReceiptNotFound is returned by GetByReceiptID when no receipt is indexed
// under the given receipt ID.
var ErrReceiptNotFound = errors.New("registry: receipt not found")

// Registry indexes receipts for lookup by receipt ID and by bundle hash.
type Registry interface {
	// Register indexes a receipt. It is an idempotent upsert keyed by receipt
	// ID. Both the receipt ID and bundle hash must be non-empty.
	Register(ctx context.Context, r receipt.Receipt) error
	// GetByReceiptID returns the receipt indexed under receiptID, or
	// ErrReceiptNotFound.
	GetByReceiptID(ctx context.Context, receiptID string) (receipt.Receipt, error)
	// FindByBundleHash returns every receipt whose bundle hash matches, in a
	// deterministic order. An empty slice (not an error) means no receipt
	// custodies that bundle.
	FindByBundleHash(ctx context.Context, bundleHash string) ([]receipt.Receipt, error)
	// List returns every indexed receipt in a deterministic order.
	List(ctx context.Context) ([]receipt.Receipt, error)
}

// ReceiptLister is the minimal read view of a receipt store the registry needs
// to (re)build its index. *receipt.FileSystemReceiptStore and
// *receipt.InMemoryReceiptStore both satisfy it.
type ReceiptLister interface {
	List(ctx context.Context) ([]receipt.Receipt, error)
}

// InMemoryRegistry is a concurrency-safe in-memory Registry.
type InMemoryRegistry struct {
	mu     sync.RWMutex
	byID   map[string]receipt.Receipt
	byHash map[string]map[string]struct{} // bundle_hash -> set of receipt IDs
}

var _ Registry = (*InMemoryRegistry)(nil)

// NewInMemoryRegistry returns an empty in-memory registry.
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		byID:   make(map[string]receipt.Receipt),
		byHash: make(map[string]map[string]struct{}),
	}
}

// Build constructs an in-memory registry pre-populated from a receipt store, so
// a daemon can index existing receipts at startup.
func Build(ctx context.Context, lister ReceiptLister) (*InMemoryRegistry, error) {
	receipts, err := lister.List(ctx)
	if err != nil {
		return nil, err
	}
	r := NewInMemoryRegistry()
	for _, rec := range receipts {
		if err := r.Register(ctx, rec); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Register indexes a receipt as an idempotent upsert keyed by receipt ID.
func (r *InMemoryRegistry) Register(ctx context.Context, rec receipt.Receipt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id := strings.TrimSpace(rec.ReceiptID)
	if id == "" {
		return errors.New("registry: receipt id required")
	}
	hash := strings.TrimSpace(rec.BundleHash)
	if hash == "" {
		return errors.New("registry: bundle hash required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// If this receipt ID was already indexed under a different bundle hash, drop
	// the stale digest mapping so the reverse index stays consistent.
	if prev, ok := r.byID[id]; ok {
		if prevHash := strings.TrimSpace(prev.BundleHash); prevHash != "" && prevHash != hash {
			r.unindexHashLocked(prevHash, id)
		}
	}

	r.byID[id] = rec
	ids, ok := r.byHash[hash]
	if !ok {
		ids = make(map[string]struct{})
		r.byHash[hash] = ids
	}
	ids[id] = struct{}{}
	return nil
}

// GetByReceiptID returns the receipt indexed under receiptID.
func (r *InMemoryRegistry) GetByReceiptID(ctx context.Context, receiptID string) (receipt.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return receipt.Receipt{}, err
	}
	id := strings.TrimSpace(receiptID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.byID[id]
	if !ok {
		return receipt.Receipt{}, ErrReceiptNotFound
	}
	return rec, nil
}

// FindByBundleHash returns every receipt whose bundle hash matches, sorted by
// submitted time then receipt ID for determinism.
func (r *InMemoryRegistry) FindByBundleHash(ctx context.Context, bundleHash string) ([]receipt.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	hash := strings.TrimSpace(bundleHash)
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byHash[hash]
	out := make([]receipt.Receipt, 0, len(ids))
	for id := range ids {
		if rec, ok := r.byID[id]; ok {
			out = append(out, rec)
		}
	}
	sortReceipts(out)
	return out, nil
}

// List returns every indexed receipt, sorted by submitted time then receipt ID.
func (r *InMemoryRegistry) List(ctx context.Context) ([]receipt.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]receipt.Receipt, 0, len(r.byID))
	for _, rec := range r.byID {
		out = append(out, rec)
	}
	sortReceipts(out)
	return out, nil
}

// unindexHashLocked removes a receipt ID from a bundle-hash bucket, dropping the
// bucket entirely when it becomes empty. The caller must hold r.mu.
func (r *InMemoryRegistry) unindexHashLocked(hash, id string) {
	ids, ok := r.byHash[hash]
	if !ok {
		return
	}
	delete(ids, id)
	if len(ids) == 0 {
		delete(r.byHash, hash)
	}
}

// sortReceipts orders receipts by submitted time then receipt ID so listings
// and digest lookups are deterministic.
func sortReceipts(receipts []receipt.Receipt) {
	sort.Slice(receipts, func(i, j int) bool {
		if receipts[i].SubmittedAt != receipts[j].SubmittedAt {
			return receipts[i].SubmittedAt < receipts[j].SubmittedAt
		}
		return receipts[i].ReceiptID < receipts[j].ReceiptID
	})
}
