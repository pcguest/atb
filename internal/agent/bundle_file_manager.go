// SPDX-License-Identifier: MIT
//
// Production agent runtime uses BundleFileManager: real .atb files via
// internal/bundle. MemoryBundleManager is the in-memory test double.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/pkg/custody"
)

// agentRawEventType wraps capture API events as opaque JSON without claiming a
// profile-specific event shape. Profile mapping is deferred to later prompts.
const agentRawEventType = "ai.agent.event.recorded"

type fileSessionRecord struct {
	params     OpenParams
	bundlePath string
	bundle     *bundle.Bundle
	eventCount int
	createdAt  time.Time
	closed     bool
	closedAt   time.Time
}

// BundleFileManager implements BundleManager with on-disk hash-chained bundles.
type BundleFileManager struct {
	dataDir  string
	mu       sync.RWMutex
	sessions map[SessionID]*fileSessionRecord
	now      func() time.Time
}

// NewBundleFileManager constructs a disk-backed session manager rooted at dataDir.
func NewBundleFileManager(dataDir string) *BundleFileManager {
	return &BundleFileManager{
		dataDir:  dataDir,
		sessions: make(map[SessionID]*fileSessionRecord),
		now:      time.Now,
	}
}

// OpenSession creates or resumes a bundle at the session path and persists it.
func (m *BundleFileManager) OpenSession(ctx context.Context, params OpenParams) (SessionID, error) {
	id, err := newSessionID()
	if err != nil {
		return "", err
	}

	bundlePath := strings.TrimSpace(params.BundlePath)
	if bundlePath == "" {
		bundlePath = sessionBundlePath(m.dataDir, id)
	}

	var (
		b          *bundle.Bundle
		createdAt  time.Time
		eventCount int
	)
	if _, statErr := os.Stat(bundlePath); statErr == nil {
		b, err = bundle.LoadVerified(bundlePath)
		if err != nil {
			return "", fmt.Errorf("agent: open existing bundle: %w", err)
		}
		createdAt = manifestCreatedAt(b)
		eventCount = nonManifestRecordCount(b)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("agent: stat bundle path: %w", statErr)
	} else {
		b, err = bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV2})
		if err != nil {
			return "", fmt.Errorf("agent: new bundle: %w", err)
		}
		createdAt = m.now().UTC()
		if err := b.Save(ctx, bundlePath); err != nil {
			return "", fmt.Errorf("agent: save new bundle: %w", err)
		}
	}

	record := &fileSessionRecord{
		params:     params,
		bundlePath: bundlePath,
		bundle:     b,
		eventCount: eventCount,
		createdAt:  createdAt,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = record
	return id, nil
}

// AppendEvent writes a raw agent wrapper record and persists the bundle.
func (m *BundleFileManager) AppendEvent(ctx context.Context, sessionID SessionID, event PendingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	if record.closed {
		return ErrSessionClosed
	}

	data, err := pendingEventData(event)
	if err != nil {
		return err
	}

	opts := appendOptionsForSession(record.params)
	if err := record.bundle.AppendWithOptions(agentRawEventType, data, opts); err != nil {
		return fmt.Errorf("agent: append event: %w", err)
	}
	if err := record.bundle.Save(ctx, record.bundlePath); err != nil {
		return fmt.Errorf("agent: save bundle: %w", err)
	}
	record.eventCount++
	return nil
}

// CloseSession marks the session closed and returns bundle metadata from disk state.
func (m *BundleFileManager) CloseSession(ctx context.Context, sessionID SessionID, _ CloseSessionOpts) (BundleMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.sessions[sessionID]
	if !ok {
		return BundleMetadata{}, ErrSessionNotFound
	}
	if record.closed {
		return BundleMetadata{}, ErrSessionClosed
	}

	if err := record.bundle.Save(ctx, record.bundlePath); err != nil {
		return BundleMetadata{}, fmt.Errorf("agent: save bundle on close: %w", err)
	}

	verified, err := bundle.LoadVerified(record.bundlePath)
	if err != nil {
		return BundleMetadata{}, fmt.Errorf("agent: verify bundle on close: %w", err)
	}

	closedAt := m.now().UTC()
	record.closed = true
	record.closedAt = closedAt
	record.bundle = verified

	meta := BundleMetadata{
		SessionID:  sessionID,
		Path:       record.bundlePath,
		ProfileID:  record.params.ProfileID,
		HeadHash:   custody.HeadHash(verified),
		EventCount: record.eventCount,
		CreatedAt:  record.createdAt,
		ClosedAt:   closedAt,
	}
	if err := writeSessionMeta(m.dataDir, meta); err != nil {
		return BundleMetadata{}, err
	}
	return meta, nil
}

// Shutdown discards in-memory session handles. Bundle files remain on disk.
func (m *BundleFileManager) Shutdown(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[SessionID]*fileSessionRecord)
	return nil
}

// ActiveSessionCount returns open session handles. Used by tests.
func (m *BundleFileManager) ActiveSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

func sessionBundlePath(dataDir string, sessionID SessionID) string {
	return filepath.Join(dataDir, "sessions", sessionID.String(), "bundle.atb")
}

func appendOptionsForSession(params OpenParams) *bundle.AppendOptions {
	actorID := strings.TrimSpace(params.ActorID)
	if actorID == "" {
		return nil
	}
	return &bundle.AppendOptions{ActorID: &actorID}
}

// pendingEventData maps capture PendingEvent into the raw agent event payload.
func pendingEventData(event PendingEvent) (map[string]any, error) {
	data := map[string]any{
		"source_event_type": strings.TrimSpace(event.EventType),
	}
	payload := strings.TrimSpace(event.Payload)
	if payload == "" {
		return data, nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		data["payload_raw"] = event.Payload
		return data, nil
	}
	data["payload"] = decoded
	return data, nil
}

func nonManifestRecordCount(b *bundle.Bundle) int {
	if b == nil {
		return 0
	}
	n := 0
	for _, rec := range b.Records {
		if rec.Event.Type != bundle.ManifestEventType {
			n++
		}
	}
	return n
}

func manifestCreatedAt(b *bundle.Bundle) time.Time {
	if b == nil || len(b.Records) == 0 {
		return time.Time{}
	}
	if b.Records[0].Event.Type != bundle.ManifestEventType {
		return time.Time{}
	}
	ts := strings.TrimSpace(b.Records[0].Event.Timestamp)
	if ts == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
