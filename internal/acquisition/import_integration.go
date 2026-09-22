// SPDX-License-Identifier: MIT
package acquisition

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/capture"
	"github.com/pcguest/atb/internal/event"
)

const (
	// ImportChatlogFormat identifies the chatlog import mode.
	ImportChatlogFormat = "chatlog"

	// ImportOTelFormat identifies the OTel import mode.
	ImportOTelFormat = "otel"
)

// ImportOptions configures an import operation.
type ImportOptions struct {
	// Format is the import format (chatlog, otel).
	Format string

	// InputPath is the path to the input file, or "-" for stdin.
	InputPath string

	// BundlePath is the path to the bundle file.
	BundlePath string

	// SnapshotName is an optional snapshot name to create after import.
	SnapshotName string

	// OutputFormat is the output format (text, json).
	OutputFormat string

	// MaxInputBytes limits the input size.
	MaxInputBytes int64

	// Continue enables continuation from a previous checkpoint.
	Continue bool

	// Reconcile enables reconciliation mode for re-import.
	Reconcile bool

	// CheckpointPath overrides the default checkpoint path.
	CheckpointPath string

	// SourceIdentity overrides the source identity for the import.
	SourceIdentity event.SourceIdentity
}

// ImportResult holds the result of an import operation.
type ImportResult struct {
	EventsWritten    int
	SkippedRecords   int
	ReconciledCount  int
	NewCount         int
	ChangedCount     int
	UnchangedCount   int
	UnknownCount     int
	BundlePath       string
	SnapshotAppended bool
	SnapshotName     string
}

// cappedReader limits the total bytes read from an io.Reader.
type cappedReader struct {
	r   io.Reader
	n   int64
	max int64
}

func (c *cappedReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.max {
		return n, fmt.Errorf("input exceeds maximum size (%d bytes)", c.max)
	}
	return n, err
}

// ImportChatlog imports a chatlog file into an ATB bundle with acquisition continuity.
func ImportChatlog(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	if opts.BundlePath == "" {
		return nil, fmt.Errorf("bundle path required")
	}
	if opts.InputPath == "" {
		return nil, fmt.Errorf("input path required")
	}

	// Determine checkpoint path
	checkpointPath := opts.CheckpointPath
	if checkpointPath == "" {
		checkpointPath = ResolveCheckpointPath(opts.BundlePath, "chatlog", opts.InputPath)
	}

	// Ensure checkpoint directory exists
	if err := EnsureCheckpointDir(opts.BundlePath); err != nil {
		return nil, fmt.Errorf("checkpoint dir: %w", err)
	}

	// Load or create checkpoint manager
	sourceIdentity := event.SourceIdentity{
		System:   "chatlog",
		RecordID: opts.InputPath,
		Derived:  true,
	}
	if opts.SourceIdentity.System != "" || opts.SourceIdentity.RecordID != "" {
		sourceIdentity = opts.SourceIdentity
	}

	cm := NewCheckpointManager(
		checkpointPath,
		"chatlog",
		opts.InputPath,
		"atb.chatlog.generic-jsonl",
		"1.0.0",
		sourceIdentity,
	)

	if err := cm.LoadOrInitialize(); err != nil {
		return nil, fmt.Errorf("checkpoint: %w", err)
	}

	// Validate source if continuing
	if opts.Continue || opts.Reconcile {
		if err := cm.ValidateSource(); err != nil {
			return nil, fmt.Errorf("checkpoint validation failed: %w", err)
		}
	}

	// Read and parse input
	var rawReader io.Reader
	if opts.InputPath == "-" {
		rawReader = os.Stdin
	} else {
		file, err := os.Open(filepath.Clean(opts.InputPath))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("input file not found: %s", opts.InputPath)
			}
			return nil, fmt.Errorf("cannot open input file: %w", err)
		}
		defer file.Close()
		rawReader = file
	}

	// Apply size cap
	cr := &cappedReader{r: rawReader, max: opts.MaxInputBytes}
	reader := cr

	// Parse chatlog
	messages, err := capture.ParseChatlog(capture.FormatGenericJSONL, reader)
	if err != nil {
		return nil, fmt.Errorf("parse chatlog: %w", err)
	}

	// Map messages to events
	mapped, err := capture.MapMessagesToEvents(messages)
	if err != nil {
		return nil, fmt.Errorf("map messages: %w", err)
	}

	// Load or create bundle
	b, created, err := loadSnapshotBundle(ctx, opts.BundlePath, false)
	if err != nil {
		return nil, fmt.Errorf("load bundle: %w", err)
	}
	if created {
		if err := stampManifestProvenance(b, "bundle_provenance", bundle.BundleProvenanceRetrospective); err != nil {
			return nil, fmt.Errorf("manifest provenance: %w", err)
		}
	}

	// Reconciliation
	_ = NewReconciler(nil)
	if opts.Continue || opts.Reconcile {
		// Load known records from existing bundle
		// TODO: Implement loading known records from bundle
	}

	// Process events with reconciliation
	_ = &ImportResult{BundlePath: opts.BundlePath}
	written := 0

	for i, spec := range mapped.Events {
		// Determine source identity for this event
		sourceIdentity := event.SourceIdentity{
			System:   "chatlog",
			RecordID: fmt.Sprintf("line:%d", i),
			Derived:  true,
		}

		// Reconciliation
		if opts.Reconcile {
			// Build acquisition info for this event
			acquisition := spec.Acquisition
			if acquisition == nil {
				acquisition = buildChatlogAcquisitionInfo(i, spec.Timestamp, opts.InputPath)
			}

			// Get source digest from acquisition
			sourceDigest := ""
			if acquisition != nil {
				sourceDigest = acquisition.SourceDigest
			}

			// Reconcile
			outcome, err := ReconcileRecord(opts.Reconcile, spec.Acquisition, sourceDigest, sourceIdentity)
			if err != nil {
				return nil, fmt.Errorf("reconcile: %w", err)
			}

			switch outcome.Result {
			case ReconciliationUnchanged:
				// Skip duplicate
				continue
			case ReconciliationChanged:
				// Record finding
				if outcome.Finding != nil {
					// TODO: Add finding to bundle/incident
				}
			case ReconciliationNew:
				// Proceed to append
			case ReconciliationUnknown:
				// Treat as new
			}
		}

		// Append event
		if err := b.AppendWithOptions(spec.Type, spec.Data, &bundle.AppendOptions{
			Timestamp:   spec.Timestamp,
			Acquisition: spec.Acquisition,
		}); err != nil {
			return nil, fmt.Errorf("append event: %w", err)
		}
		written++
	}

	// Update checkpoint
	// TODO: Update checkpoint with latest position

	// Save bundle
	if err := b.Save(opts.BundlePath); err != nil {
		return nil, fmt.Errorf("save bundle: %w", err)
	}

	// Save checkpoint
	// TODO: Save checkpoint

	return &ImportResult{
		EventsWritten: written,
		BundlePath:    opts.BundlePath,
	}, nil
}

// ImportOTel imports an OTLP/JSON trace export into an ATB bundle with acquisition continuity.
func ImportOTel(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	// Similar structure to ImportChatlog but for OTel
	return nil, fmt.Errorf("not yet implemented")
}

// buildChatlogAcquisitionInfo creates AcquisitionInfo for a chatlog record.
func buildChatlogAcquisitionInfo(line int, timestamp, inputPath string) *event.AcquisitionInfo {
	acquiredAt := time.Now().UTC().Format(time.RFC3339Nano)
	sourceRecordID := fmt.Sprintf("line:%d", line)

	return &event.AcquisitionInfo{
		Mode:            "retrospective",
		SourceSystem:    "chatlog",
		SourceRecordID:  sourceRecordID,
		SourceTimestamp: "",
		AcquiredAt:      acquiredAt,
		SourceDigest:    "",
		Adapter:         "atb.chatlog.generic-jsonl",
		AdapterVersion:  "1.0.0",
		Checkpoint: &event.CheckpointInfo{
			SourceSystem:      "chatlog",
			AcquisitionStream: "stdin",
			Position:          fmt.Sprintf("line:%d", line),
			ObservedAt:        acquiredAt,
			Adapter:           "atb.chatlog.generic-jsonl",
			AdapterVersion:    "1.0.0",
		},
	}
}

// ReconcileRecord reconciles a record against known records.
func ReconcileRecord(reconcile bool, acquisition *event.AcquisitionInfo, sourceDigest string, sourceIdentity event.SourceIdentity) (*ReconciliationOutcome, error) {
	// This is a placeholder - actual reconciliation happens in ImportChatlog
	return nil, fmt.Errorf("not implemented")
}

// loadSnapshotBundle loads or creates a bundle for snapshot operations.
func loadSnapshotBundle(ctx context.Context, bundlePath string, created bool) (*bundle.Bundle, bool, error) {
	// This is a placeholder - the actual implementation is in cmd/atb/import.go
	return nil, false, fmt.Errorf("not implemented")
}

// stampManifestProvenance stamps the manifest with provenance information.
func stampManifestProvenance(b *bundle.Bundle, key, value string) error {
	// This is a placeholder - the actual implementation is in cmd/atb/import.go
	return nil
}
