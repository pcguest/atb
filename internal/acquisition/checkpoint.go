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

// ErrCheckpointBundleMismatch indicates the checkpoint was written against a
// different committed bundle head (or record count) than the one now loaded.
// This makes a stale or foreign checkpoint fail closed instead of silently
// resuming as if it described the current evidence.
var ErrCheckpointBundleMismatch = errors.New("acquisition: checkpoint bundle binding mismatch")

// ErrNoTranslatableSpans indicates an OTLP payload contained no translatable spans.
var ErrNoTranslatableSpans = errors.New("acquisition: no translatable spans found in OTLP payload")

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

	// BundleHeadHash is the record hash of the committed bundle head at the time
	// this checkpoint was written. Empty for legacy checkpoints. When set,
	// continuation fails closed if the loaded bundle head does not match, so a
	// checkpoint can never be used as if it described different evidence.
	BundleHeadHash string `json:"bundle_head_hash,omitempty"`

	// BundleRecordCount is the committed bundle record count at the time this
	// checkpoint was written.
	BundleRecordCount int `json:"bundle_record_count,omitempty"`

	// Metadata for future extensibility.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Save writes the checkpoint to the given path durably.
//
// It writes a uniquely-named temporary file in the destination directory, fsyncs
// it, renames it over the destination, then fsyncs the parent directory. A crash
// mid-write cannot leave a truncated checkpoint, and two concurrent writers
// cannot clobber a shared `path + ".tmp"`. The checkpoint is operational state,
// not evidence: this only hardens its persistence.
func (c *Checkpoint) Save(path string) error {
	if c.FormatVersion == 0 {
		c.FormatVersion = CheckpointFormatVersion
	}

	cleanPath := filepath.Clean(path)
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("checkpoint: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("checkpoint: marshal: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(cleanPath)+".*.tmp") // #nosec G304 -- checkpoint path is a local operator-selected path; checkpoint state is operational, not evidence.
	if err != nil {
		return fmt.Errorf("checkpoint: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("checkpoint: chmod temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("checkpoint: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("checkpoint: fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("checkpoint: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, cleanPath); err != nil {
		return fmt.Errorf("checkpoint: rename: %w", err)
	}
	removeTemp = false

	if err := syncDir(dir); err != nil {
		return fmt.Errorf("checkpoint: fsync parent: %w", err)
	}
	return nil
}

// Load loads a checkpoint from the given path.
func Load(path string) (*Checkpoint, error) {
	// #nosec G304 -- --checkpoint is an intentional local operator-selected path; checkpoint state is operational, not remotely request-controlled evidence.
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

// BindBundle records the committed bundle head and record count this checkpoint
// describes. Callers must set this only after the bundle has been durably saved,
// so a checkpoint can never name a head that is not yet committed.
func (c *Checkpoint) BindBundle(headHash string, recordCount int) {
	c.BundleHeadHash = headHash
	c.BundleRecordCount = recordCount
}

// ValidateBundle fails closed when the checkpoint was bound to a different
// committed bundle head (or record count) than the one now loaded. A legacy
// checkpoint with no binding (empty head) is accepted for backward
// compatibility; a bound checkpoint must match exactly.
func (c *Checkpoint) ValidateBundle(headHash string, recordCount int) error {
	if c.BundleHeadHash == "" {
		return nil
	}
	if c.BundleHeadHash != headHash || c.BundleRecordCount != recordCount {
		return fmt.Errorf("%w: checkpoint bound to head %s (%d records), but loaded bundle head is %s (%d records)",
			ErrCheckpointBundleMismatch, c.BundleHeadHash, c.BundleRecordCount, headHash, recordCount)
	}
	return nil
}
