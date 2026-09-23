// SPDX-License-Identifier: MIT
package acquisition

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pcguest/atb/internal/event"
)

// CheckpointManager manages checkpoint persistence and loading for an acquisition session.
type CheckpointManager struct {
	// CheckpointPath is the path to the checkpoint file.
	CheckpointPath string

	// Checkpoint is the current checkpoint state.
	Checkpoint *Checkpoint

	// SourceSystem identifies the source system.
	SourceSystem string

	// AcquisitionStream identifies the acquisition stream.
	AcquisitionStream string

	// Adapter identifies the adapter being used.
	Adapter string

	// AdapterVersion is the adapter version.
	AdapterVersion string

	// SourceIdentity is the source identity for this acquisition.
	SourceIdentity event.SourceIdentity
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager(checkpointPath, sourceSystem, acquisitionStream, adapter, adapterVersion string, sourceIdentity event.SourceIdentity) *CheckpointManager {
	return &CheckpointManager{
		CheckpointPath:    checkpointPath,
		SourceSystem:      sourceSystem,
		AcquisitionStream: acquisitionStream,
		Adapter:           adapter,
		AdapterVersion:    adapterVersion,
		SourceIdentity:    sourceIdentity,
	}
}

// LoadOrInitialize loads an existing checkpoint or creates a new one.
func (cm *CheckpointManager) LoadOrInitialize() error {
	cp, err := LoadOrCreate(cm.CheckpointPath, cm.SourceSystem, cm.AcquisitionStream, cm.Adapter, cm.SourceIdentity)
	if err != nil {
		return fmt.Errorf("checkpoint: load or create: %w", err)
	}
	cm.Checkpoint = cp
	return nil
}

// ValidateSource validates that the checkpoint matches the expected source.
func (cm *CheckpointManager) ValidateSource() error {
	if cm.Checkpoint == nil {
		return fmt.Errorf("checkpoint not loaded")
	}
	return cm.Checkpoint.Validate(cm.SourceSystem, cm.AcquisitionStream, cm.Adapter)
}

// UpdateCheckpoint updates the checkpoint with the latest position and digest.
func (cm *CheckpointManager) UpdateCheckpoint(position, lastRecordDigest string, processedCount int) {
	if cm.Checkpoint == nil {
		return
	}
	cm.Checkpoint.UpdatePosition(position, lastRecordDigest, processedCount)
}

// Save persists the checkpoint to disk.
func (cm *CheckpointManager) Save() error {
	if cm.Checkpoint == nil {
		return fmt.Errorf("checkpoint not initialized")
	}
	return cm.Checkpoint.Save(cm.CheckpointPath)
}

// ResolveCheckpointPath resolves the checkpoint path based on bundle path and source.
func ResolveCheckpointPath(bundlePath, sourceSystem, acquisitionStream string) string {
	baseName := filepath.Base(bundlePath)
	ext := filepath.Ext(baseName)
	name := baseName[:len(baseName)-len(ext)]

	checkpointName := fmt.Sprintf("%s.%s.checkpoint.json", name, sanitizeStreamName(acquisitionStream))
	return filepath.Join(filepath.Dir(bundlePath), CheckpointDir, checkpointName)
}

// sanitizeStreamName reduces an acquisition stream (often a caller-supplied file
// path) to a safe single filename component. It strips directory components and
// any character that could traverse or alter the checkpoint path, and appends a
// short digest so distinct streams with the same basename cannot collide.
func sanitizeStreamName(stream string) string {
	base := filepath.Base(stream)
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	name := strings.Trim(b.String(), "._-")
	if name == "" {
		name = "stream"
	}
	if len(name) > 40 {
		name = name[:40]
	}
	sum := sha256.Sum256([]byte(stream))
	return name + "-" + hex.EncodeToString(sum[:4])
}

// EnsureCheckpointDir ensures the checkpoint directory exists.
func EnsureCheckpointDir(bundlePath string) error {
	dir := filepath.Join(filepath.Dir(bundlePath), CheckpointDir)
	return os.MkdirAll(dir, 0750)
}
