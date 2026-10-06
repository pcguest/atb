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
	CheckpointPath     string
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
	if opts.Continue || opts.Reconcile {
		if !created {
			if err := b.Verify(); err != nil {
				return nil, fmt.Errorf("existing bundle failed integrity verification; refusing to reconcile onto an unverified chain: %w", err)
			}
		}
		// Validate the checkpoint binding even when the bundle was just created:
		// a surviving bound checkpoint with no matching bundle must fail closed
		// rather than be silently rebound to a new empty bundle.
		if err := cm.ValidateBundle(committedBundleHead(b), len(b.Records)); err != nil {
			return nil, fmt.Errorf("checkpoint bundle binding: %w", err)
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
	plan, err := planReconciliation(events, reconciler, opts.Reconcile || opts.Continue)
	if err != nil {
		return nil, err
	}
	if plan.writesAcquisition() {
		if err := requireAcquisitionProfile(b, created); err != nil {
			return nil, err
		}
	}

	counts, err := applyReconciliation(b, plan)
	if err != nil {
		return nil, err
	}
	written := counts.written

	// Save the bundle durably before the checkpoint, so a checkpoint can never
	// name a head that is not yet committed.
	if err := b.Save(opts.BundlePath); err != nil {
		return nil, fmt.Errorf("save bundle: %w", err)
	}

	// Update and bind the checkpoint to the committed bundle head.
	cm.UpdateCheckpoint(fmt.Sprintf("exchange:%d", uniqueSourceRecords(events)), "", written)
	cm.BindBundle(committedBundleHead(b), len(b.Records))
	if err := cm.Save(); err != nil {
		return nil, fmt.Errorf("save checkpoint: %w", err)
	}

	return &ImportResult{
		// SkippedRecords counts every source record that produced no evidence in
		// this pass: records the mapper deliberately dropped (e.g. system turns)
		// plus records reconciliation found already committed. Reporting only the
		// reconciliation count would let mapper-drop silently under-report.
		EventsWritten:      written,
		SkippedRecords:     mapped.SkippedRecords + counts.unchanged,
		ReconciledCount:    counts.written,
		NewCount:           counts.newCount,
		ChangedCount:       counts.changed,
		UnchangedCount:     counts.unchanged,
		UnknownCount:       counts.unknown,
		BundlePath:         opts.BundlePath,
		CheckpointPath:     checkpointPath,
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
	if opts.Continue || opts.Reconcile {
		if !created {
			if err := b.Verify(); err != nil {
				return nil, fmt.Errorf("existing bundle failed integrity verification; refusing to reconcile onto an unverified chain: %w", err)
			}
		}
		// Validate the checkpoint binding even when the bundle was just created:
		// a surviving bound checkpoint with no matching bundle must fail closed
		// rather than be silently rebound to a new empty bundle.
		if err := cm.ValidateBundle(committedBundleHead(b), len(b.Records)); err != nil {
			return nil, fmt.Errorf("checkpoint bundle binding: %w", err)
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
	plan, err := planReconciliation(events, reconciler, opts.Reconcile || opts.Continue)
	if err != nil {
		return nil, err
	}
	if plan.writesAcquisition() {
		if err := requireAcquisitionProfile(b, created); err != nil {
			return nil, err
		}
	}

	counts, err := applyReconciliation(b, plan)
	if err != nil {
		return nil, err
	}
	written := counts.written

	// Save the bundle durably before the checkpoint, so a checkpoint can never
	// name a head that is not yet committed.
	if err := b.Save(opts.BundlePath); err != nil {
		return nil, fmt.Errorf("save bundle: %w", err)
	}

	// Update and bind the checkpoint to the committed bundle head.
	cm.UpdateCheckpoint(fmt.Sprintf("span:%d", len(result.Events)), "", written)
	cm.BindBundle(committedBundleHead(b), len(b.Records))
	if err := cm.Save(); err != nil {
		return nil, fmt.Errorf("save checkpoint: %w", err)
	}

	return &ImportResult{
		// SkippedRecords includes spans the translator deliberately dropped
		// (ErrUnsupported) plus records reconciliation found already committed.
		EventsWritten:      written,
		SkippedRecords:     result.SkippedCount + counts.unchanged,
		ReconciledCount:    counts.written,
		NewCount:           counts.newCount,
		ChangedCount:       counts.changed,
		UnchangedCount:     counts.unchanged,
		UnknownCount:       counts.unknown,
		BundlePath:         opts.BundlePath,
		CheckpointPath:     checkpointPath,
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

// committedBundleHead returns the record hash of the last committed bundle
// record, used to bind a checkpoint to the exact evidence it describes. It is
// identity, not integrity: callers that care about validity verify separately.
func committedBundleHead(b *bundle.Bundle) string {
	if b == nil || len(b.Records) == 0 {
		return ""
	}
	return b.Records[len(b.Records)-1].Hash
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

// reconcileAction is one planned append: the event to write and, for a CHANGED
// source record, the single bounded finding that accompanies it.
type reconcileAction struct {
	event   importEvent
	finding *FindingInfo
	skip    bool
}

// reconciliationPlan is the side-effect-free result of reconciling an import
// against an existing bundle. Planning never mutates the bundle; records are
// only appended when the plan is applied.
type reconciliationPlan struct {
	actions []reconcileAction
	counts  reconcileCounts
}

// writesAcquisition reports whether applying the plan would append any
// acquisition-bearing event. The manifest-version floor is enforced only when
// this is true, so an import that reconciles every source record as UNCHANGED
// (writing nothing) is never refused for targeting a pre-v3 bundle.
func (p reconciliationPlan) writesAcquisition() bool {
	for i := range p.actions {
		if !p.actions[i].skip && p.actions[i].event.acquisition != nil {
			return true
		}
	}
	return false
}

// planReconciliation decides, per source record, whether each event is appended
// or skipped.
//
// A source record is identified by its acquisition SourceIdentity. Every event
// derived from the same source record shares one decision: UNCHANGED skips the
// whole record (no duplicate evidence); CHANGED emits exactly one finding and
// appends the new representation; NEW/UNKNOWN append. This makes re-import
// idempotent for unchanged sources and bounded for changed ones.
func planReconciliation(events []importEvent, reconciler *Reconciler, reconcile bool) (reconciliationPlan, error) {
	var plan reconciliationPlan
	type decision struct {
		result  ReconciliationResult
		digest  string
		skip    bool
		finding *FindingInfo
	}
	decisions := make(map[string]*decision)
	plan.actions = make([]reconcileAction, 0, len(events))

	for i := range events {
		ev := events[i]
		acq := ev.acquisition
		hasID := acq != nil && acq.SourceRecordID != ""
		key := fmt.Sprintf("__unknown__:%d", i)
		if hasID {
			key = acq.SourceSystem + ":" + acq.SourceRecordID
		}

		d, ok := decisions[key]
		action := reconcileAction{event: ev}
		switch {
		case !ok:
			d = &decision{}
			if reconcile && hasID {
				outcome, err := reconciler.Reconcile(
					event.SourceIdentity{System: acq.SourceSystem, RecordID: acq.SourceRecordID, Derived: true},
					acq.SourceDigest, acq.AcquiredAt, acq,
				)
				if err != nil {
					return reconciliationPlan{}, fmt.Errorf("reconcile: %w", err)
				}
				d.result = outcome.Result
				d.finding = outcome.Finding
				d.digest = acq.SourceDigest
				d.skip = outcome.Result == ReconciliationUnchanged
			} else if reconcile {
				d.result = ReconciliationUnknown
			}
			if hasID {
				d.digest = acq.SourceDigest
			}
			decisions[key] = d
			action.skip = d.skip
			switch d.result {
			case ReconciliationNew:
				plan.counts.newCount++
			case ReconciliationChanged:
				plan.counts.changed++
			case ReconciliationUnchanged:
				plan.counts.unchanged++
			case ReconciliationUnknown:
				plan.counts.unknown++
			}
		case reconcile && hasID && acq.SourceDigest != d.digest:
			// The same source identity appears again within this pass with a
			// different representation. Surface one changed-source finding and
			// append the new representation; never silently collapse the change.
			action.finding = &FindingInfo{
				Flag:           "source_record_changed",
				Severity:       "high",
				Title:          "Source record changed within a single acquisition pass",
				SourceRecordID: acq.SourceRecordID,
				PreviousDigest: d.digest,
				CurrentDigest:  acq.SourceDigest,
				AcquiredAt:     acq.AcquiredAt,
			}
			action.skip = false
			d.digest = acq.SourceDigest
			d.finding = nil
		case reconcile && hasID:
			// Same identity and identical representation repeated within one
			// pass: idempotent replay, so never append it twice.
			action.skip = true
		default:
			// Non-reconcile (plain import) keeps historical append semantics.
			action.skip = d.skip
		}

		if d.finding != nil {
			action.finding = d.finding
			d.finding = nil // emit once per source record
		}
		plan.actions = append(plan.actions, action)
	}
	return plan, nil
}

// applyReconciliation appends the events selected by a plan to the bundle.
func applyReconciliation(b *bundle.Bundle, plan reconciliationPlan) (reconcileCounts, error) {
	counts := plan.counts
	for i := range plan.actions {
		action := plan.actions[i]
		if action.skip {
			continue
		}
		if action.finding != nil {
			if err := appendAcquisitionFinding(b, action.finding, action.event.acquisition); err != nil {
				return counts, fmt.Errorf("append finding: %w", err)
			}
		}
		if err := b.AppendWithOptions(action.event.typ, action.event.data, &bundle.AppendOptions{
			Timestamp:    action.event.timestamp,
			TraceID:      action.event.traceID,
			SpanID:       action.event.spanID,
			ParentSpanID: action.event.parentSpanID,
			Acquisition:  action.event.acquisition,
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
			// Create a new bundle that declares the acquisition-aware canonical
			// profile (manifest v3). Imported events always carry acquisition
			// provenance, so a reader that predates the acquisition envelope
			// must reject the bundle loudly rather than mis-hash it.
			b, err = bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV3})
			if err != nil {
				return nil, false, err
			}
			return b, true, nil
		}
		return nil, false, fmt.Errorf("load bundle: %w", err)
	}
	return b, false, nil
}

// requireAcquisitionProfile enforces the manifest-version floor for bundles that
// carry acquisition provenance. A freshly created import bundle declares v3; an
// existing bundle that predates v3 (manifest v1/v2) must not receive
// acquisition-bearing events, because a reader that predates the acquisition
// envelope would silently drop the field and report a spurious tamper. Refuse
// rather than create a mixed-version chain. Callers invoke this only when the
// plan would actually append acquisition, so a no-op reconciliation of an
// unchanged source is never refused.
func requireAcquisitionProfile(b *bundle.Bundle, created bool) error {
	if created {
		return nil
	}
	m := b.Manifest()
	if m != nil && m.Version == "3" {
		return nil
	}
	return fmt.Errorf("cannot append acquisition-bearing events to a bundle that declares manifest version < 3; re-import into a new bundle so the acquisition canonical profile is declared")
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
