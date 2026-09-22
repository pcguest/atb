// SPDX-License-Identifier: MIT
package acquisition

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/event"
)

const (
	// CheckpointFormatVersion is the current checkpoint format version.
	CheckpointFormatVersion = 1

	// CheckpointFileName is the default checkpoint file name.
	CheckpointFileName = "acquisition.checkpoint.json"

	// CheckpointDir is the default directory for checkpoints.
	CheckpointDir = ".atb/checkpoints"
)

// ErrCheckpointNotFound indicates a checkpoint file was not found.
var ErrCheckpointNotFound = errors.New("acquisition: checkpoint not found")

// ErrCheckpointVersionUnsupported indicates an unsupported checkpoint format version.
var ErrCheckpointVersionUnsupported = errors.New("acquisition: unsupported checkpoint format version")

// ErrCheckpointCorrupted indicates a checkpoint file is malformed.
var ErrCheckpointCorrupted = errors.New("acquisition: checkpoint corrupted")

// ErrCheckpointSourceMismatch indicates the checkpoint source doesn't match.
var ErrCheckpointSourceMismatch = errors.New("acquisition: checkpoint source mismatch")

// ErrCheckpointAdapterMismatch indicates the checkpoint adapter doesn't match.
var ErrCheckpointAdapterMismatch = errors.New("acquisition: checkpoint adapter mismatch")

// Checkpoint represents an acquisition checkpoint for incremental continuation.
type Checkpoint struct {
	// FormatVersion is the checkpoint format version.
	FormatVersion int `json:"format_version"`

	// SourceSystem identifies the source system (e.g., "chatlog", "otel").
	SourceSystem string `json:"source_system"`

	// AcquisitionStream identifies the acquisition stream (e.g., file path, OTel endpoint).
	AcquisitionStream string `json:"acquisition_stream"`

	// Position is the position/watermark in the source stream (e.g., byte offset, line number, cursor).
	Position string `json:"position"`

	// ObservedAt is the RFC 3339 timestamp when this checkpoint was observed.
	ObservedAt string `json:"observed_at"`

	// Adapter identifies the adapter used for this checkpoint.
	Adapter string `json:"adapter"`

	// AdapterVersion is the version of the adapter used.
	AdapterVersion string `json:"adapter_version,omitempty"`

	// SourceIdentity is the stable source identity for deduplication.
	SourceIdentity event.SourceIdentity `json:"source_identity"`

	// ProcessedCount is the number of records processed up to this checkpoint.
	ProcessedCount int `json:"processed_count"`

	// LastRecordDigest is the digest of the last processed record.
	LastRecordDigest string `json:"last_record_digest,omitempty"`

	// Metadata for future extensibility.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Save writes the checkpoint to the given path atomically.
// The checkpoint is written to a temporary file and then renamed atomically.
func (c *Checkpoint) Save(path string) error {
	if c.FormatVersion == 0 {
		c.FormatVersion = CheckpointFormatVersion
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("checkpoint: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("checkpoint: marshal: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("checkpoint: write temp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("checkpoint: rename: %w", err)
	}

	return nil
}

// Load loads a checkpoint from the given path.
func Load(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrCheckpointNotFound
		}
		return nil, fmt.Errorf("checkpoint: read: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCheckpointCorrupted, err)
	}

	if cp.FormatVersion == 0 {
		cp.FormatVersion = 1 // default for legacy
	}

	if cp.FormatVersion > 1 {
		return nil, fmt.Errorf("%w: got %d, max supported is 1", ErrCheckpointVersionUnsupported, cp.FormatVersion)
	}

	return &cp, nil
}

// LoadOrCreate loads an existing checkpoint or creates a new one if it doesn't exist.
func LoadOrCreate(path string, sourceSystem, acquisitionStream, adapter string, sourceIdentity event.SourceIdentity) (*Checkpoint, error) {
	cp, err := Load(path)
	if err != nil {
		if errors.Is(err, ErrCheckpointNotFound) {
			cp = &Checkpoint{
				FormatVersion:     1,
				SourceSystem:      sourceSystem,
				AcquisitionStream: acquisitionStream,
				Adapter:           adapter,
				SourceIdentity:    sourceIdentity,
				ObservedAt:        time.Now().UTC().Format(time.RFC3339Nano),
				Position:          "start",
			}
			return cp, nil
		}
		return nil, err
	}
	return cp, nil
}

// Validate validates the checkpoint against expected source and adapter.
func (c *Checkpoint) Validate(sourceSystem, acquisitionStream, adapter string) error {
	if c.SourceSystem != sourceSystem {
		return fmt.Errorf("%w: expected %q, got %q", ErrCheckpointSourceMismatch, sourceSystem, c.SourceSystem)
	}
	if c.AcquisitionStream != acquisitionStream {
		return fmt.Errorf("%w: expected %q, got %q", ErrCheckpointSourceMismatch, acquisitionStream, c.AcquisitionStream)
	}
	if c.Adapter != adapter {
		return fmt.Errorf("%w: expected %q, got %q", ErrCheckpointAdapterMismatch, adapter, c.Adapter)
	}
	return nil
}

// UpdatePosition updates the checkpoint position and related metadata.
func (c *Checkpoint) UpdatePosition(position string, lastRecordDigest string, processedCount int) {
	c.Position = position
	c.LastRecordDigest = lastRecordDigest
	c.ProcessedCount = processedCount
	c.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
}
