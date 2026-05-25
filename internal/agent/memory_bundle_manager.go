// SPDX-License-Identifier: MIT
package agent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type sessionRecord struct {
	params    OpenParams
	bundlePath string
	events    []PendingEvent
	createdAt time.Time
	closed    bool
	closedAt  time.Time
}

// MemoryBundleManager is a test-only stub: placeholder paths and synthetic
// hashes. Production uses BundleFileManager (bundle_file_manager.go).
type MemoryBundleManager struct {
	dataDir  string
	mu       sync.RWMutex
	sessions map[SessionID]*sessionRecord
	now      func() time.Time
}

// NewMemoryBundleManager constructs an in-memory session manager rooted at
// dataDir for placeholder bundle paths.
func NewMemoryBundleManager(dataDir string) *MemoryBundleManager {
	return &MemoryBundleManager{
		dataDir:  dataDir,
		sessions: make(map[SessionID]*sessionRecord),
		now:      time.Now,
	}
}

// OpenSession registers a new session and returns its identifier.
func (m *MemoryBundleManager) OpenSession(_ context.Context, params OpenParams) (SessionID, error) {
	id, err := newSessionID()
	if err != nil {
		return "", err
	}

	bundlePath := strings.TrimSpace(params.BundlePath)
	if bundlePath == "" {
		bundlePath = filepath.Join(m.dataDir, "sessions", id.String(), "bundle.atb")
	}

	record := &sessionRecord{
		params:     params,
		bundlePath: bundlePath,
		createdAt:  m.now().UTC(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = record
	return id, nil
}

// AppendEvent queues a placeholder event on an open session.
func (m *MemoryBundleManager) AppendEvent(_ context.Context, sessionID SessionID, event PendingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	if record.closed {
		return ErrSessionClosed
	}
	record.events = append(record.events, event)
	return nil
}

// CloseSession marks a session closed and returns synthetic metadata.
func (m *MemoryBundleManager) CloseSession(_ context.Context, sessionID SessionID, _ CloseSessionOpts) (BundleMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.sessions[sessionID]
	if !ok {
		return BundleMetadata{}, ErrSessionNotFound
	}
	if record.closed {
		return BundleMetadata{}, ErrSessionClosed
	}

	closedAt := m.now().UTC()
	record.closed = true
	record.closedAt = closedAt

	return BundleMetadata{
		SessionID:  sessionID,
		Path:       record.bundlePath,
		ProfileID:  record.params.ProfileID,
		HeadHash:   syntheticHeadHash(record.events),
		EventCount: len(record.events),
		CreatedAt:  record.createdAt,
		ClosedAt:   closedAt,
	}, nil
}

// Shutdown discards all tracked sessions.
func (m *MemoryBundleManager) Shutdown(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[SessionID]*sessionRecord)
	return nil
}

// ActiveSessionCount returns the number of tracked sessions. Used by tests.
func (m *MemoryBundleManager) ActiveSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

func newSessionID() (SessionID, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("agent: generate session id: %w", err)
	}
	return SessionID("sess_" + hex.EncodeToString(buf[:])), nil
}

func syntheticHeadHash(events []PendingEvent) string {
	if len(events) == 0 {
		return "sha256:" + hex.EncodeToString(make([]byte, sha256.Size))
	}
	last := events[len(events)-1]
	sum := sha256.Sum256([]byte(last.EventType + "|" + last.Payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}
