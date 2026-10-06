// SPDX-License-Identifier: MIT
package proxy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/acquisition"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/capturejournal"
	"github.com/pcguest/atb/internal/event"
)

const (
	// captureSourceSystem identifies intercept as the acquisition source.
	captureSourceSystem = "atb.proxy"
	// captureAdapter identifies the intercept adapter.
	captureAdapter = "atb.intercept"
	// captureAdapterVersion is the intercept adapter version.
	captureAdapterVersion = "1.0.0"
	// captureRepresentationVersion identifies the digested representation.
	captureRepresentationVersion = "atb.intercept.event.v1"
	// captureJournalDir is the journal directory relative to the bundle dir.
	captureJournalDir = ".atb/capture"
)

// captureState is the operator-facing continuity state of the capture path.
type captureState string

const (
	// captureHealthy means the journal and evidence commit are consistent.
	captureHealthy captureState = "healthy"
	// captureRecoveryRequired means uncommitted durable observations exist.
	captureRecoveryRequired captureState = "recovery_required"
	// captureDegraded means capture cannot be trusted to be continuous.
	captureDegraded captureState = "degraded"
	// captureUnknown means continuity could not be established.
	captureUnknown captureState = "unknown"
)

// captureCoordinator owns the durable journal, the checkpoint, and the single
// commit protocol for the intercept path. It is not safe for concurrent use;
// the BundleRecorder mutex is the single writer.
type captureCoordinator struct {
	bundlePath  string
	streamID    string
	incarnation string
	journal     *capturejournal.Journal
	cm          *acquisition.CheckpointManager
	// fresh reports whether no checkpoint existed at startup, so a bundle that
	// already holds acquisition evidence cannot be resumed without operational
	// continuity.
	fresh bool

	// Health / operator state.
	state             captureState
	stateDetail       string
	lastObservationAt string
	lastCommitAt      string
	replayed          int
	lastErr           error
	knownGap          bool
}

// observationPayload is the exact representation digested and journaled for one
// observation. Replay reconstructs the ATB event from this representation, so
// the live and replay materialisations are byte-identical.
type observationPayload struct {
	Type         string          `json:"type"`
	Data         json.RawMessage `json:"data"`
	ActorID      *string         `json:"actor_id,omitempty"`
	OrgID        *string         `json:"org_id,omitempty"`
	WorkspaceID  *string         `json:"workspace_id,omitempty"`
	Timestamp    string          `json:"timestamp,omitempty"`
	TraceID      string          `json:"trace_id,omitempty"`
	SpanID       string          `json:"span_id,omitempty"`
	ParentSpanID string          `json:"parent_span_id,omitempty"`
}

// enableCapture opens (or creates) the journal and checkpoint for the bundle,
// establishes the source incarnation, and runs startup recovery. It must be
// called before the first observation is appended.
func (r *BundleRecorder) enableCapture(sourceIncarnation string) error {
	if r == nil {
		return errors.New("proxy: nil recorder")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.capture != nil && !r.captureStopped {
		return nil
	}
	r.capture = nil
	r.captureStopped = false

	bundlePath := r.path
	streamID := filepath.Base(bundlePath)
	journalPath := filepath.Join(filepath.Dir(bundlePath), captureJournalDir, streamID+".journal.ndjson")
	checkpointPath := acquisition.ResolveCheckpointPath(bundlePath, captureSourceSystem, streamID)

	// Determine whether a checkpoint already exists and its incarnation.
	existing, err := acquisition.Load(checkpointPath)
	fresh := errors.Is(err, acquisition.ErrCheckpointNotFound)
	if err != nil && !fresh {
		return fmt.Errorf("capture: load checkpoint: %w", err)
	}

	journal, err := capturejournal.Open(journalPath, streamID)
	if err != nil {
		return fmt.Errorf("capture: open journal: %w", err)
	}

	incarnation := strings.TrimSpace(sourceIncarnation)
	if incarnation == "" {
		switch {
		case !fresh && existing.SourceIncarnation != "":
			incarnation = existing.SourceIncarnation
		case journalIncarnation(journal) != "":
			// A lost checkpoint but a surviving journal: adopt the incarnation
			// the journal was written with rather than inventing a new source.
			incarnation = journalIncarnation(journal)
		default:
			incarnation, err = newIncarnation()
			if err != nil {
				_ = journal.Close()
				return err
			}
		}
	}

	cm := acquisition.NewCheckpointManager(
		checkpointPath,
		captureSourceSystem,
		streamID,
		captureAdapter,
		captureAdapterVersion,
		event.SourceIdentity{System: captureSourceSystem, RecordID: streamID, Derived: true},
	)
	cm.SourceIncarnation = incarnation
	if err := cm.LoadOrInitialize(); err != nil {
		_ = journal.Close()
		return fmt.Errorf("capture: load checkpoint: %w", err)
	}
	if !fresh {
		// Fails closed on a wrong source, adapter version, or incarnation.
		if err := cm.ValidateSource(); err != nil {
			_ = journal.Close()
			return fmt.Errorf("capture: checkpoint validation: %w", err)
		}
	}
	// Bind the incarnation onto a checkpoint that predates incarnation tracking,
	// so a later different token cannot silently resume the same source.
	if cm.Checkpoint.SourceIncarnation == "" {
		cm.Checkpoint.SourceIncarnation = incarnation
	}

	c := &captureCoordinator{
		bundlePath:  bundlePath,
		streamID:    streamID,
		incarnation: incarnation,
		journal:     journal,
		cm:          cm,
		fresh:       fresh,
		state:       captureUnknown,
	}
	if journal.Repaired() {
		c.knownGap = true
		c.stateDetail = "journal torn tail repaired on open; observations may be incomplete"
	}
	if err := c.recover(); err != nil {
		_ = journal.Close()
		return err
	}
	r.capture = c
	return nil
}

// recover establishes continuity and replays any durable-but-uncommitted
// observations. It never starts fresh silently: a bound checkpoint that does
// not match the loaded bundle, a checkpoint ahead of the journal, or a journal
// entry inconsistent with this capture all fail closed.
func (c *captureCoordinator) recover() error {
	b, created, err := loadBundleForCapture(c.bundlePath)
	if err != nil {
		return fmt.Errorf("capture: load bundle: %w", err)
	}
	if !created {
		if err := b.Verify(); err != nil {
			return fmt.Errorf("capture: existing bundle failed verification: %w", err)
		}
		if m := b.Manifest(); m == nil || m.Version != "3" {
			return fmt.Errorf("capture: existing bundle is not manifest v3; start a new bundle for live capture")
		}
	}

	if err := validateCheckpointPrefix(c.cm.Checkpoint, b); err != nil {
		return fmt.Errorf("capture: checkpoint bundle binding: %w", err)
	}

	committedPos, err := checkpointPosition(c.cm.Checkpoint)
	if err != nil {
		return fmt.Errorf("capture: checkpoint position: %w", err)
	}
	if bundleHasAcquisition(b) && c.journal.Len() == 0 && committedPos == 0 {
		// The bundle already holds live evidence, but the journal and committed
		// position are gone. Resuming would silently reuse observation
		// identities and could lose or suppress distinct observations, so fail
		// closed instead of claiming continuity.
		return fmt.Errorf("%w: bundle holds live acquisition evidence but no checkpoint position or journal remains; operational continuity was lost",
			acquisition.ErrCheckpointBundleMismatch)
	}
	if c.fresh {
		// No checkpoint anchors this run. The surviving journal must fully
		// account for every committed observation; a partial journal (some
		// committed observations lost) must fail closed rather than report
		// healthy over incomplete capture.
		if err := journalAccountsForEvidence(b, c.journal); err != nil {
			return err
		}
	}
	if committedPos > c.journal.LastPosition() {
		// The checkpoint names a position the journal cannot account for.
		return fmt.Errorf("%w: checkpoint position %d is ahead of journal position %d",
			acquisition.ErrCheckpointBundleMismatch, committedPos, c.journal.LastPosition())
	}

	pending := c.journal.EntriesFrom(committedPos)
	for i := range pending {
		if err := c.validateEntry(pending[i]); err != nil {
			return err
		}
	}

	index := indexAcquisition(b)
	appended := 0
	for _, e := range pending {
		if d, ok := index[e.ObservationID]; ok {
			if d == e.RepresentationDigest {
				// Committed but the checkpoint lagged the crash; do not duplicate.
				continue
			}
			return fmt.Errorf("capture: observation %s present with a different digest", e.ObservationID)
		}
		if err := appendMaterialised(b, e, c.streamID); err != nil {
			return fmt.Errorf("capture: replay %s: %w", e.ObservationID, err)
		}
		index[e.ObservationID] = e.RepresentationDigest
		appended++
	}

	if appended > 0 {
		if err := b.Save(c.bundlePath); err != nil {
			return fmt.Errorf("capture: save recovered bundle: %w", err)
		}
	}
	// Bind only to a bundle that is actually durable. A freshly-created bundle
	// that has not been saved has a non-deterministic manifest (created_at,
	// bundle_id), so binding to it would fail closed on the next start.
	if !created || appended > 0 {
		c.cm.BindBundle(committedHead(b), len(b.Records))
	}
	if appended > 0 || !created {
		// Only advance/reset the checkpoint when recovery actually changed
		// durable state; otherwise preserve the recorded commit timestamp.
		c.cm.UpdateCheckpoint(strconv.FormatInt(c.journal.LastPosition(), 10), "", c.journal.Len())
	}
	if err := c.cm.Save(); err != nil {
		return fmt.Errorf("capture: save checkpoint: %w", err)
	}

	c.replayed = appended
	c.lastCommitAt = c.cm.Checkpoint.ObservedAt
	switch {
	case c.knownGap:
		c.state = captureDegraded
	case appended > 0:
		c.state = captureHealthy
		c.stateDetail = fmt.Sprintf("recovered %d durable observation(s) after restart", appended)
	default:
		c.state = captureHealthy
	}
	return nil
}

// commit runs the full commit protocol for one observation:
//
//	observation -> durable journal append -> materialise ATB event(s) ->
//	durable bundle save -> checkpoint bound -> journal position committed.
//
// It drains every durable-but-uncommitted journal entry, so an earlier failed
// commit can never be skipped by a later one.
func (c *captureCoordinator) commit(ev *event.Event) (string, error) {
	entry, err := c.journalObservation(ev)
	if err != nil {
		c.fail(err)
		return "", err
	}
	hash, err := c.commitPending(entry.ObservationID)
	if err != nil {
		c.fail(err)
		return "", err
	}
	c.lastErr = nil
	c.lastObservationAt = entry.ObservedAt
	c.lastCommitAt = time.Now().UTC().Format(time.RFC3339Nano)
	if c.knownGap {
		c.state = captureDegraded
	} else {
		c.state = captureHealthy
	}
	return hash, nil
}

// commitPending materialises every durable journal entry after the committed
// position, saves the bundle once, binds the checkpoint to the committed head,
// and advances the committed position. It returns the record hash for wantID.
func (c *captureCoordinator) commitPending(wantID string) (string, error) {
	committedPos, err := checkpointPosition(c.cm.Checkpoint)
	if err != nil {
		return "", err
	}
	b, created, err := loadBundleForCapture(c.bundlePath)
	if err != nil {
		return "", err
	}
	if created {
		if m := b.Manifest(); m == nil || m.Version != "3" {
			return "", errors.New("capture: new bundle did not initialise as manifest v3")
		}
	}

	index := indexAcquisition(b)
	appended := 0
	for _, e := range c.journal.EntriesFrom(committedPos) {
		if err := c.validateEntry(e); err != nil {
			return "", err
		}
		if d, ok := index[e.ObservationID]; ok {
			if d != e.RepresentationDigest {
				return "", fmt.Errorf("capture: observation %s present with a different digest", e.ObservationID)
			}
			continue
		}
		if err := appendMaterialised(b, e, c.streamID); err != nil {
			return "", err
		}
		index[e.ObservationID] = e.RepresentationDigest
		appended++
	}
	if appended > 0 {
		if err := b.Save(c.bundlePath); err != nil {
			return "", err
		}
	}

	c.cm.UpdateCheckpoint(strconv.FormatInt(c.journal.LastPosition(), 10), "", c.journal.Len())
	c.cm.BindBundle(committedHead(b), len(b.Records))
	if err := c.cm.Save(); err != nil {
		// The evidence is durable but the checkpoint lags; a later commit or
		// restart will reconcile idempotently.
		return "", err
	}

	for i := len(b.Records) - 1; i >= 0; i-- {
		if acq := b.Records[i].Event.Acquisition; acq != nil && acq.SourceRecordID == wantID {
			return b.Records[i].Hash, nil
		}
	}
	return "", fmt.Errorf("capture: materialised record for %s not found", wantID)
}

// validateEntry fails closed when a journal entry is not consistent with this
// capture's source, adapter, incarnation, or representation contract.
func (c *captureCoordinator) validateEntry(e capturejournal.Entry) error {
	if e.SourceSystem != captureSourceSystem {
		return fmt.Errorf("capture: journal entry source %q does not match %q", e.SourceSystem, captureSourceSystem)
	}
	if e.Adapter != captureAdapter {
		return fmt.Errorf("capture: journal entry adapter %q does not match %q", e.Adapter, captureAdapter)
	}
	if e.AdapterVersion != captureAdapterVersion {
		return fmt.Errorf("capture: journal entry adapter version %q does not match %q", e.AdapterVersion, captureAdapterVersion)
	}
	if e.RepresentationVersion != captureRepresentationVersion {
		return fmt.Errorf("capture: journal entry representation version %q does not match %q", e.RepresentationVersion, captureRepresentationVersion)
	}
	if e.SourceIncarnation != "" && e.SourceIncarnation != c.incarnation {
		return fmt.Errorf("capture: journal entry incarnation %q does not match %q", e.SourceIncarnation, c.incarnation)
	}
	if err := verifyEntryDigest(e); err != nil {
		return err
	}
	return nil
}

// verifyEntryDigest fails closed when a journal entry's recorded representation
// digest does not match the representation it carries, so a self-asserted
// provenance digest can never be bound to a different payload.
func verifyEntryDigest(e capturejournal.Entry) error {
	got, err := digestRepresentation(e.Payload)
	if err != nil {
		return fmt.Errorf("capture: recompute digest for %s: %w", e.ObservationID, err)
	}
	if got != e.RepresentationDigest {
		return fmt.Errorf("%w: journal entry %s representation digest does not match its payload",
			acquisition.ErrCheckpointCorrupted, e.ObservationID)
	}
	return nil
}

// bundleHasAcquisition reports whether any record carries live acquisition
// provenance.
func bundleHasAcquisition(b *bundle.Bundle) bool {
	if b == nil {
		return false
	}
	for _, rec := range b.Records {
		if rec.Event.Acquisition != nil {
			return true
		}
	}
	return false
}

// journalAccountsForEvidence fails closed when the bundle holds committed
// acquisition evidence that the surviving journal cannot account for. It is
// used when no checkpoint anchors the run, so a partially-lost journal cannot
// be mistaken for a complete one.
func journalAccountsForEvidence(b *bundle.Bundle, j *capturejournal.Journal) error {
	if !bundleHasAcquisition(b) {
		return nil
	}
	present := map[string]bool{}
	for _, e := range j.Entries() {
		present[e.ObservationID] = true
	}
	for _, rec := range b.Records {
		acq := rec.Event.Acquisition
		if acq == nil || acq.SourceRecordID == "" {
			continue
		}
		if !present[acq.SourceRecordID] {
			return fmt.Errorf("%w: no checkpoint and the journal does not account for committed observation %q; operational continuity was lost",
				acquisition.ErrCheckpointBundleMismatch, acq.SourceRecordID)
		}
	}
	return nil
}

func (c *captureCoordinator) fail(err error) {
	c.lastErr = err
	c.state = captureDegraded
}

// journalObservation durably journals one event and returns its entry.
func (c *captureCoordinator) journalObservation(ev *event.Event) (capturejournal.Entry, error) {
	payload, err := marshalObservation(ev)
	if err != nil {
		return capturejournal.Entry{}, err
	}
	digest, err := digestRepresentation(payload)
	if err != nil {
		return capturejournal.Entry{}, err
	}
	pos := c.journal.LastPosition() + 1
	// The observation identity is scoped by incarnation as well as position, so
	// a fresh incarnation can never reuse a prior run's identity and silently
	// suppress a distinct observation.
	observationID := fmt.Sprintf("%s/%s:%d", c.incarnation, c.streamID, pos)
	return c.journal.Append(capturejournal.Entry{
		ObservationID:         observationID,
		SourceSystem:          captureSourceSystem,
		SourceIncarnation:     c.incarnation,
		RepresentationVersion: captureRepresentationVersion,
		RepresentationDigest:  digest,
		ObservationType:       ev.Type,
		Payload:               payload,
		SourceTimestamp:       ev.Timestamp,
		ObservedAt:            time.Now().UTC().Format(time.RFC3339Nano),
		Adapter:               captureAdapter,
		AdapterVersion:        captureAdapterVersion,
	})
}

// Health returns the current capture error, if any.
func (c *captureCoordinator) Health() error {
	if c == nil {
		return nil
	}
	return c.lastErr
}

// EnableCapture turns on the continuous-capture journal and commit protocol for
// this recorder. It runs startup recovery and must be called before the first
// observation is appended.
func (r *BundleRecorder) EnableCapture(sourceIncarnation string) error {
	return r.enableCapture(sourceIncarnation)
}

// closeCapture releases the journal file handle. It must be called when the
// recorder is stopped: on Windows an open handle prevents the file from being
// removed or rotated.
func (r *BundleRecorder) closeCapture() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.capture != nil && r.capture.journal != nil {
		_ = r.capture.journal.Close()
	}
	// Keep r.capture set but mark the recorder stopped, so a late append is
	// refused instead of falling back to the unjournalled legacy path.
	r.captureStopped = true
}

// Close releases capture resources (the journal file handle). It is safe to
// call when capture is disabled and is intended for callers that own the
// recorder directly (e.g. tests and embedders); Proxy.Stop calls it
// internally.
func (r *BundleRecorder) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.capture != nil && r.capture.journal != nil {
		return r.capture.journal.Close()
	}
	return nil
}

// CaptureHealth reports the last capture durability error, if any. It is nil
// when capture is disabled or healthy.
func (r *BundleRecorder) CaptureHealth() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.capture == nil {
		return nil
	}
	return r.capture.Health()
}

// journalIncarnation returns the source incarnation recorded on the first
// journal entry, or "" when the journal is empty or has none.
func journalIncarnation(j *capturejournal.Journal) string {
	if j == nil {
		return ""
	}
	entries := j.Entries()
	if len(entries) == 0 {
		return ""
	}
	return strings.TrimSpace(entries[0].SourceIncarnation)
}

func newIncarnation() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("capture: generate incarnation: %w", err)
	}
	return "inc-" + hex.EncodeToString(b[:]), nil
}

func marshalObservation(ev *event.Event) (json.RawMessage, error) {
	if ev == nil {
		return nil, errors.New("capture: nil event")
	}
	data, err := json.Marshal(ev.Data)
	if err != nil {
		return nil, fmt.Errorf("capture: marshal event data: %w", err)
	}
	p := observationPayload{
		Type:         ev.Type,
		Data:         data,
		ActorID:      ev.ActorID,
		OrgID:        ev.OrgID,
		WorkspaceID:  ev.WorkspaceID,
		Timestamp:    ev.Timestamp,
		TraceID:      ev.TraceID,
		SpanID:       ev.SpanID,
		ParentSpanID: ev.ParentSpanID,
	}
	return json.Marshal(p)
}

func digestRepresentation(payload json.RawMessage) (string, error) {
	canonical, err := canonicalize.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("capture: canonicalize observation: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// appendMaterialised rebuilds the ATB event from a journal entry and appends it
// with live acquisition provenance on a v3 bundle.
func appendMaterialised(b *bundle.Bundle, e capturejournal.Entry, streamID string) error {
	var p observationPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return fmt.Errorf("capture: decode observation: %w", err)
	}
	digest, err := digestRepresentation(e.Payload)
	if err != nil {
		return fmt.Errorf("capture: recompute digest for %s: %w", e.ObservationID, err)
	}
	if digest != e.RepresentationDigest {
		return fmt.Errorf("%w: journal entry %s representation digest does not match its payload",
			acquisition.ErrCheckpointCorrupted, e.ObservationID)
	}
	var data any
	if len(p.Data) > 0 {
		if err := json.Unmarshal(p.Data, &data); err != nil {
			return fmt.Errorf("capture: decode observation data: %w", err)
		}
	}
	acq := &event.AcquisitionInfo{
		Mode:            "live",
		SourceSystem:    e.SourceSystem,
		SourceRecordID:  e.ObservationID,
		SourceTimestamp: e.SourceTimestamp,
		AcquiredAt:      e.ObservedAt,
		SourceDigest:    digest,
		Adapter:         e.Adapter,
		AdapterVersion:  e.AdapterVersion,
		Checkpoint: &event.CheckpointInfo{
			SourceSystem:      e.SourceSystem,
			AcquisitionStream: streamID,
			Position:          strconv.FormatInt(e.Position, 10),
			ObservedAt:        e.ObservedAt,
			Adapter:           e.Adapter,
			AdapterVersion:    e.AdapterVersion,
		},
	}
	opts := &bundle.AppendOptions{
		Timestamp:    p.Timestamp,
		ActorID:      p.ActorID,
		OrgID:        p.OrgID,
		WorkspaceID:  p.WorkspaceID,
		TraceID:      p.TraceID,
		SpanID:       p.SpanID,
		ParentSpanID: p.ParentSpanID,
		Acquisition:  acq,
	}
	if err := b.AppendWithOptions(p.Type, data, opts); err != nil {
		return err
	}
	return nil
}

func loadBundleForCapture(path string) (*bundle.Bundle, bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			b, err := bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV3})
			if err != nil {
				return nil, false, err
			}
			return b, true, nil
		}
		return nil, false, err
	}
	b, err := bundle.LoadVerified(path)
	if err != nil {
		return nil, false, err
	}
	return b, false, nil
}

// validateCheckpointPrefix enforces the invariant that a checkpoint never
// describes evidence that is not durably present. Unlike an exact head match,
// it accepts a bundle that is legitimately AHEAD of the checkpoint (evidence
// was committed but the checkpoint lagged a crash), while still failing closed
// when the checkpoint is ahead of the evidence, is internally inconsistent, or
// names a head that is not a prefix of the current bundle.
func validateCheckpointPrefix(cp *acquisition.Checkpoint, b *bundle.Bundle) error {
	if cp == nil {
		return nil
	}
	if cp.BundleHeadHash == "" && cp.BundleRecordCount == 0 {
		return nil
	}
	if cp.BundleHeadHash == "" || cp.BundleRecordCount <= 0 {
		return fmt.Errorf("%w: checkpoint has an inconsistent bundle binding (head %q, count %d)",
			acquisition.ErrCheckpointBundleMismatch, cp.BundleHeadHash, cp.BundleRecordCount)
	}
	if cp.BundleRecordCount > len(b.Records) {
		return fmt.Errorf("%w: checkpoint describes %d records but bundle has %d",
			acquisition.ErrCheckpointBundleMismatch, cp.BundleRecordCount, len(b.Records))
	}
	if b.Records[cp.BundleRecordCount-1].Hash != cp.BundleHeadHash {
		return fmt.Errorf("%w: checkpoint head is not a prefix of the current bundle",
			acquisition.ErrCheckpointBundleMismatch)
	}
	return nil
}

// checkpointPosition parses the committed journal position. A missing or
// unset ("start") position is zero; any other malformed value fails closed.
func checkpointPosition(cp *acquisition.Checkpoint) (int64, error) {
	if cp == nil {
		return 0, nil
	}
	pos := strings.TrimSpace(cp.Position)
	if pos == "" || pos == "start" {
		return 0, nil
	}
	n, err := strconv.ParseInt(pos, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%w: invalid checkpoint position %q", acquisition.ErrCheckpointCorrupted, pos)
	}
	return n, nil
}

func committedHead(b *bundle.Bundle) string {
	if b == nil || len(b.Records) == 0 {
		return ""
	}
	return b.Records[len(b.Records)-1].Hash
}

// indexAcquisition maps source_record_id -> source_digest for every committed
// acquisition-bearing record, so replay can skip already-committed observations.
func indexAcquisition(b *bundle.Bundle) map[string]string {
	out := map[string]string{}
	if b == nil {
		return out
	}
	for _, rec := range b.Records {
		if rec.Event.Acquisition == nil {
			continue
		}
		id := rec.Event.Acquisition.SourceRecordID
		if id == "" {
			continue
		}
		out[id] = rec.Event.Acquisition.SourceDigest
	}
	return out
}
