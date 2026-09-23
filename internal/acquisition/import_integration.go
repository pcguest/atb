// SPDX-License-Identifier: MIT
package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/capture"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/hash"
	"github.com/pcguest/atb/pkg/otel"
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

	// Stdin is an optional reader for stdin input (used when InputPath is "-").
	// If nil, os.Stdin is used.
	Stdin io.Reader
}

// ImportResult holds the result of an import operation.
type ImportResult struct {
	EventsWritten      int
	SkippedRecords     int
	ReconciledCount    int
	NewCount           int
	ChangedCount       int
	UnchangedCount     int
	UnknownCount       int
	BundlePath         string
	SnapshotAppended   bool
	SnapshotName       string
	UnknownTurnIndices []int
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
		if opts.Stdin != nil {
			rawReader = opts.Stdin
		} else {
			rawReader = os.Stdin
		}
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
	messages, err := capture.ParseChatlog(opts.Format, reader)
	if err != nil {
		return nil, fmt.Errorf("parse chatlog: %w", err)
	}

	// Map messages to events, handling unknown turns
	var unknownTurnIndices []int
	mapped, err := capture.MapMessagesToEvents(messages)
	if errors.Is(err, capture.ErrUnknownTurn) {
		var unknown []int
		messages, unknown = capture.FilterKnownTurns(messages)
		unknownTurnIndices = unknown
		mapped, err = capture.MapMessagesToEvents(messages)
		if err == nil {
			mapped.SkippedRecords += len(unknown)
		}
	}
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
	reconciler := NewReconciler(nil)
	if opts.Continue || opts.Reconcile {
		// Load known records from existing bundle
		if err := reconciler.LoadFromBundle(recordsToInterfaces(b.Records)); err != nil {
			return nil, fmt.Errorf("load known records: %w", err)
		}
	}

	events := make([]importEvent, 0, len(mapped.Events))
	for _, spec := range mapped.Events {
		events = append(events, importEvent{
			typ:         spec.Type,
			data:        spec.Data,
			timestamp:   spec.Timestamp,
			acquisition: spec.Acquisition,
		})
	}
	stampAcquisitionStream(events, opts.InputPath)

	counts, err := appendReconciled(b, events, reconciler, opts.Reconcile)
	if err != nil {
		return nil, err
	}
	written := counts.written

	// Update checkpoint
	cm.UpdateCheckpoint(fmt.Sprintf("exchange:%d", uniqueSourceRecords(events)), "", written)

	// Save bundle
	if err := b.Save(opts.BundlePath); err != nil {
		return nil, fmt.Errorf("save bundle: %w", err)
	}

	// Save checkpoint
	if err := cm.Save(); err != nil {
		return nil, fmt.Errorf("save checkpoint: %w", err)
	}

	return &ImportResult{
		EventsWritten:      written,
		SkippedRecords:     counts.unchanged + len(unknownTurnIndices),
		ReconciledCount:    counts.written,
		NewCount:           counts.newCount,
		ChangedCount:       counts.changed,
		UnchangedCount:     counts.unchanged,
		UnknownCount:       counts.unknown,
		BundlePath:         opts.BundlePath,
		SnapshotAppended:   false,
		SnapshotName:       "",
		UnknownTurnIndices: unknownTurnIndices,
	}, nil
}

// ImportOTel imports an OTLP/JSON trace export into an ATB bundle with acquisition continuity.
func ImportOTel(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	if opts.BundlePath == "" {
		return nil, fmt.Errorf("bundle path required")
	}
	if opts.InputPath == "" {
		return nil, fmt.Errorf("input path required")
	}

	// Determine checkpoint path
	checkpointPath := opts.CheckpointPath
	if checkpointPath == "" {
		checkpointPath = ResolveCheckpointPath(opts.BundlePath, "otel", opts.InputPath)
	}

	// Ensure checkpoint directory exists
	if err := EnsureCheckpointDir(opts.BundlePath); err != nil {
		return nil, fmt.Errorf("checkpoint dir: %w", err)
	}

	// Load or create checkpoint manager
	sourceIdentity := event.SourceIdentity{
		System:   "otel",
		RecordID: opts.InputPath,
		Derived:  true,
	}
	if opts.SourceIdentity.System != "" || opts.SourceIdentity.RecordID != "" {
		sourceIdentity = opts.SourceIdentity
	}

	cm := NewCheckpointManager(
		checkpointPath,
		"otel",
		opts.InputPath,
		"atb.otel.otlp-json",
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

	// Read input
	var rawReader io.Reader
	if opts.InputPath == "-" {
		if opts.Stdin != nil {
			rawReader = opts.Stdin
		} else {
			rawReader = os.Stdin
		}
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

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	// Decode and translate OTel spans
	receiver := &otel.Receiver{Translator: otel.DefaultTranslator{}}
	result, err := receiver.ReceiveJSON(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("decode/translate OTLP: %w", err)
	}
	if len(result.Events) == 0 {
		return nil, ErrNoTranslatableSpans
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
	reconciler := NewReconciler(nil)
	if opts.Continue || opts.Reconcile {
		if err := reconciler.LoadFromBundle(recordsToInterfaces(b.Records)); err != nil {
			return nil, fmt.Errorf("load known records: %w", err)
		}
	}

	events := make([]importEvent, 0, len(result.Events))
	for _, ev := range result.Events {
		events = append(events, importEvent{
			typ:          ev.Type,
			data:         ev.Data,
			timestamp:    ev.Timestamp,
			traceID:      ev.TraceID,
			spanID:       ev.SpanID,
			parentSpanID: ev.ParentSpanID,
			acquisition:  ev.Acquisition,
		})
	}
	stampAcquisitionStream(events, opts.InputPath)

	counts, err := appendReconciled(b, events, reconciler, opts.Reconcile)
	if err != nil {
		return nil, err
	}
	written := counts.written

	// Update checkpoint
	cm.UpdateCheckpoint(fmt.Sprintf("span:%d", len(result.Events)), "", written)

	// Save bundle
	if err := b.Save(opts.BundlePath); err != nil {
		return nil, fmt.Errorf("save bundle: %w", err)
	}

	// Save checkpoint
	if err := cm.Save(); err != nil {
		return nil, fmt.Errorf("save checkpoint: %w", err)
	}

	return &ImportResult{
		EventsWritten:      written,
		SkippedRecords:     counts.unchanged,
		ReconciledCount:    counts.written,
		NewCount:           counts.newCount,
		ChangedCount:       counts.changed,
		UnchangedCount:     counts.unchanged,
		UnknownCount:       counts.unknown,
		BundlePath:         opts.BundlePath,
		SnapshotAppended:   false,
		SnapshotName:       "",
		UnknownTurnIndices: nil,
	}, nil
}

// importEvent is the minimal shape needed to append one translated event.
type importEvent struct {
	typ          string
	data         any
	timestamp    string
	traceID      string
	spanID       string
	parentSpanID string
	acquisition  *event.AcquisitionInfo
}

// reconcileCounts summarises per-source-record reconciliation.
type reconcileCounts struct {
	written   int
	newCount  int
	changed   int
	unchanged int
	unknown   int
}

// stampAcquisitionStream records the real acquisition stream on each event's
// checkpoint. Capture builds acquisition before the importer knows the input
// path, so the per-record checkpoint stream would otherwise read "stdin".
func stampAcquisitionStream(events []importEvent, stream string) {
	if stream == "" {
		return
	}
	for i := range events {
		if acq := events[i].acquisition; acq != nil && acq.Checkpoint != nil {
			acq.Checkpoint.AcquisitionStream = stream
		}
	}
}

// uniqueSourceRecords counts distinct acquisition source records in events.
func uniqueSourceRecords(events []importEvent) int {
	seen := make(map[string]struct{})
	for _, ev := range events {
		if ev.acquisition != nil && ev.acquisition.SourceRecordID != "" {
			seen[ev.acquisition.SourceSystem+":"+ev.acquisition.SourceRecordID] = struct{}{}
		}
	}
	return len(seen)
}

// appendAcquisitionFinding persists one bounded acquisition finding.
func appendAcquisitionFinding(b *bundle.Bundle, f *FindingInfo, acq *event.AcquisitionInfo) error {
	data := map[string]any{
		"finding_type":         f.Flag,
		"source_system":        acq.SourceSystem,
		"source_record_id":     f.SourceRecordID,
		"previous_digest":      f.PreviousDigest,
		"current_digest":       f.CurrentDigest,
		"previous_acquired_at": f.AcquiredAtPrevious,
		"current_acquired_at":  f.AcquiredAt,
		"adapter":              acq.Adapter,
		"adapter_version":      acq.AdapterVersion,
	}
	return b.AppendWithOptions(event.TypeAcquisitionFinding, data, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// appendReconciled appends events, reconciling once per source record.
//
// A source record is identified by its acquisition SourceIdentity. Every event
// derived from the same source record shares one decision: UNCHANGED skips the
// whole record (no duplicate evidence); CHANGED emits exactly one finding and
// appends the new representation; NEW/UNKNOWN append. This makes re-import
// idempotent for unchanged sources and bounded for changed ones.
func appendReconciled(b *bundle.Bundle, events []importEvent, reconciler *Reconciler, reconcile bool) (reconcileCounts, error) {
	var counts reconcileCounts
	type decision struct {
		skip    bool
		result  ReconciliationResult
		finding *FindingInfo
		acq     *event.AcquisitionInfo
	}
	decisions := make(map[string]*decision)

	for i, ev := range events {
		acq := ev.acquisition
		key := fmt.Sprintf("__unknown__:%d", i)
		if acq != nil && acq.SourceRecordID != "" {
			key = acq.SourceSystem + ":" + acq.SourceRecordID
		}

		d, ok := decisions[key]
		if !ok {
			d = &decision{acq: acq}
			if reconcile && acq != nil && acq.SourceRecordID != "" {
				outcome, err := reconciler.Reconcile(
					event.SourceIdentity{System: acq.SourceSystem, RecordID: acq.SourceRecordID, Derived: true},
					acq.SourceDigest, acq.AcquiredAt, acq,
				)
				if err != nil {
					return counts, fmt.Errorf("reconcile: %w", err)
				}
				d.result = outcome.Result
				d.finding = outcome.Finding
				d.skip = outcome.Result == ReconciliationUnchanged
			} else if reconcile {
				d.result = ReconciliationUnknown
			}
			decisions[key] = d
			switch d.result {
			case ReconciliationNew:
				counts.newCount++
			case ReconciliationChanged:
				counts.changed++
			case ReconciliationUnchanged:
				counts.unchanged++
			case ReconciliationUnknown:
				counts.unknown++
			}
		}

		if d.skip {
			continue
		}

		if d.finding != nil {
			if err := appendAcquisitionFinding(b, d.finding, d.acq); err != nil {
				return counts, fmt.Errorf("append finding: %w", err)
			}
			d.finding = nil // emit once per source record
		}

		if err := b.AppendWithOptions(ev.typ, ev.data, &bundle.AppendOptions{
			Timestamp:    ev.timestamp,
			TraceID:      ev.traceID,
			SpanID:       ev.spanID,
			ParentSpanID: ev.parentSpanID,
			Acquisition:  ev.acquisition,
		}); err != nil {
			return counts, fmt.Errorf("append event: %w", err)
		}
		counts.written++
	}
	return counts, nil
}

// recordsToInterfaces converts []bundle.Record to []interface{}
func recordsToInterfaces(records []bundle.Record) []interface{} {
	result := make([]interface{}, len(records))
	for i, r := range records {
		result[i] = r
	}
	return result
}

// loadSnapshotBundle loads or creates a bundle for snapshot operations.
func loadSnapshotBundle(ctx context.Context, bundlePath string, created bool) (*bundle.Bundle, bool, error) {
	b, err := bundle.Load(bundlePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Create new bundle
			b, err = bundle.New()
			if err != nil {
				return nil, false, err
			}
			return b, true, nil
		}
		return nil, false, fmt.Errorf("load bundle: %w", err)
	}
	return b, false, nil
}

// stampManifestProvenance stamps the manifest with provenance information.
func stampManifestProvenance(b *bundle.Bundle, key, value string) error {
	if b == nil || len(b.Records) == 0 {
		return fmt.Errorf("bundle is empty")
	}
	if b.Records[0].Event.Type != bundle.ManifestEventType {
		return fmt.Errorf("first record is not a manifest")
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("manifest metadata key is empty")
	}

	switch data := b.Records[0].Event.Data.(type) {
	case map[string]any:
		meta, _ := data["metadata"].(map[string]any)
		if meta == nil {
			meta = map[string]any{}
		}
		meta[key] = value
		data["metadata"] = meta
		b.Records[0].Event.Data = data
	case string:
		var payload map[string]any
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return fmt.Errorf("parse manifest payload: %w", err)
		}
		meta, _ := payload["metadata"].(map[string]any)
		if meta == nil {
			meta = map[string]any{}
		}
		meta[key] = value
		payload["metadata"] = meta
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode manifest payload: %w", err)
		}
		b.Records[0].Event.Data = string(raw)
	default:
		return fmt.Errorf("manifest payload type %T is not supported", data)
	}

	// Recompute the manifest hash after modifying the data
	newHash, err := hash.Compute(b.Records[0].Event)
	if err != nil {
		return fmt.Errorf("recompute manifest hash: %w", err)
	}
	b.Records[0].Hash = newHash
	return nil
}
