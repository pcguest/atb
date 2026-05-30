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

// Store stores bundle bytes in memory. The receiptID is derived from the bundleHash.
func (s *InMemoryWORMStore) Store(ctx context.Context, bundleBytes []byte, bundleHash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	receiptID := fmt.Sprintf("sha256-%s", bundleHash)
	if _, exists := s.bundles[receiptID]; exists {
		return "", fmt.Errorf("bundle with hash %s already exists", bundleHash)
	}
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
