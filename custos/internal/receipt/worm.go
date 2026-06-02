package receipt

import (
	"context"
	"fmt"
	"sync"
)

// InMemoryWORMStore is a simple in-memory implementation of WORMStore for testing.
type InMemoryWORMStore struct {
	mu      sync.Mutex
	bundles map[string][]byte // key is bundleHash, value is raw bundle bytes
}

// NewInMemoryWORMStore creates a new InMemoryWORMStore.
func NewInMemoryWORMStore() *InMemoryWORMStore {
	return &InMemoryWORMStore{
		bundles: make(map[string][]byte),
	}
}

// Store stores bundle bytes in memory, content-addressed by bundleHash. Storing
// identical content again is idempotent (returns the existing receipt ID with
// no error), matching FileSystemWORMStore so the two implementations share one
// contract.
func (s *InMemoryWORMStore) Store(ctx context.Context, bundleBytes []byte, bundleHash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	receiptID := fmt.Sprintf("sha256-%s", bundleHash)
	s.bundles[receiptID] = bundleBytes
	return receiptID, nil
}

// Retrieve retrieves bundle bytes from memory by receiptID.
func (s *InMemoryWORMStore) Retrieve(ctx context.Context, receiptID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bundleBytes, ok := s.bundles[receiptID]
	if !ok {
		return nil, fmt.Errorf("bundle with receipt ID %s not found", receiptID)
	}
	return bundleBytes, nil
}
