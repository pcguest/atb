// SPDX-License-Identifier: MIT
// Package capturejournal implements the durable, append-only observation
// journal used by continuous capture (atb intercept).
//
// The journal is operational state, not evidence. Its purpose is to make an
// observation durable *before* it is materialised into ATB evidence, so a
// collector crash between observing a source event and committing ATB evidence
// cannot silently lose the observation. The ATB bundle remains the evidence
// substrate and the root of trust.
//
// The journal is append-only NDJSON. Each entry is hash-chained with the same
// rule as ATB evidence: SHA-256(UTF-8(hex(prev_hash)) || RFC8785(entry)). A
// torn trailing line (a partial write with no terminating newline) is the only
// recoverable corruption: Open repairs it to the last complete line and reports
// it. Any interior corruption fails closed.
package capturejournal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/hash"
)

const (
	// JournalFormatVersion is the current journal entry format version.
	JournalFormatVersion = 1

	// JournalFileName is the default journal file name.
	JournalFileName = "capture.journal.ndjson"

	// maxLineBytes bounds a single journal line. It mirrors the bundle's
	// per-record ceiling so a hostile observation cannot exhaust memory.
	maxLineBytes = 16 * 1024 * 1024

	// entryType is the synthetic event type used for the journal hash chain.
	entryType = "atb.capture.journal.entry"
)

// Entry is one durable observation.
type Entry struct {
	FormatVersion         int             `json:"format_version"`
	Position              int64           `json:"position"`
	PrevHash              string          `json:"prev_hash"`
	ObservationID         string          `json:"observation_id"`
	SourceSystem          string          `json:"source_system"`
	SourceIncarnation     string          `json:"source_incarnation,omitempty"`
	RepresentationVersion string          `json:"representation_version"`
	RepresentationDigest  string          `json:"representation_digest"`
	ObservationType       string          `json:"observation_type"`
	Payload               json.RawMessage `json:"payload"`
	SourceTimestamp       string          `json:"source_timestamp,omitempty"`
	ObservedAt            string          `json:"observed_at"`
	Adapter               string          `json:"adapter"`
	AdapterVersion        string          `json:"adapter_version"`
	Hash                  string          `json:"hash"`
}

// hashable returns the portion of an entry covered by its chain hash. Hash and
// FormatVersion are excluded so the hash can be computed before assignment.
func (e Entry) hashable() entryHashable {
	return entryHashable{
		Position:              e.Position,
		ObservationID:         e.ObservationID,
		SourceSystem:          e.SourceSystem,
		SourceIncarnation:     e.SourceIncarnation,
		RepresentationVersion: e.RepresentationVersion,
		RepresentationDigest:  e.RepresentationDigest,
		ObservationType:       e.ObservationType,
		Payload:               e.Payload,
		SourceTimestamp:       e.SourceTimestamp,
		ObservedAt:            e.ObservedAt,
		Adapter:               e.Adapter,
		AdapterVersion:        e.AdapterVersion,
	}
}

type entryHashable struct {
	Position              int64           `json:"position"`
	ObservationID         string          `json:"observation_id"`
	SourceSystem          string          `json:"source_system"`
	SourceIncarnation     string          `json:"source_incarnation,omitempty"`
	RepresentationVersion string          `json:"representation_version"`
	RepresentationDigest  string          `json:"representation_digest"`
	ObservationType       string          `json:"observation_type"`
	Payload               json.RawMessage `json:"payload"`
	SourceTimestamp       string          `json:"source_timestamp,omitempty"`
	ObservedAt            string          `json:"observed_at"`
	Adapter               string          `json:"adapter"`
	AdapterVersion        string          `json:"adapter_version"`
}

func computeHash(e Entry) (string, error) {
	return hash.Compute(hash.Event{
		Sequence: int(e.Position),
		PrevHash: e.PrevHash,
		Type:     entryType,
		Data:     e.hashable(),
	})
}

// Journal is an open append-only observation journal.
type Journal struct {
	path     string
	streamID string
	f        *os.File
	entries  []Entry
	repaired bool
	// failed is set when a write or fsync error occurred. Once failed, the
	// journal refuses further appends: the in-memory tail no longer matches the
	// durable file, so continuing would reuse a position and corrupt the chain.
	failed bool
}

// Open opens or creates the journal at path for the given stream identifier,
// verifies the existing chain, and repairs a torn trailing line if present.
func Open(path, streamID string) (*Journal, error) {
	clean := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(clean), 0o750); err != nil {
		return nil, fmt.Errorf("capturejournal: mkdir: %w", err)
	}
	_, statErr := os.Stat(clean)
	created := os.IsNotExist(statErr)
	if created {
		// A brand-new journal at this path is not the journal a stale repair
		// marker describes; drop the repair marker so a fresh journal is not
		// reported as torn. A degradation marker is deliberately NOT dropped: it
		// can only have been written after a journal existed, so a marker beside
		// a missing journal means the journal was lost, which must stay
		// disclosed rather than be reported healthy on a fresh start.
		_ = os.Remove(repairMarkerPath(clean))
	}

	entries, repaired, err := readEntries(clean)
	if err != nil {
		return nil, err
	}
	if repaired {
		// Record the repair durably BEFORE truncating, so a crash cannot leave
		// a clean journal with no degradation signal. A repair is a real
		// continuity signal, not a transient one.
		if err := writeRepairMarker(clean, discardedTailBytes(clean)); err != nil {
			return nil, fmt.Errorf("capturejournal: persist repair marker: %w", err)
		}
		// Truncate the torn trailing partial line so appends start clean.
		if err := os.Truncate(clean, lastCompleteOffset(clean)); err != nil {
			return nil, fmt.Errorf("capturejournal: repair torn tail: %w", err)
		}
	}
	// A prior repair (marker present) keeps the journal in a degraded state even
	// though the file itself is now clean.
	if markerPresent(clean) {
		repaired = true
	}

	f, err := os.OpenFile(clean, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- operator-selected local journal path
	if err != nil {
		return nil, fmt.Errorf("capturejournal: open: %w", err)
	}
	if created {
		// Make the new file and any newly created parent directories durable
		// before the journal can hold observations.
		if err := syncDirChain(filepath.Dir(clean)); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("capturejournal: fsync dir: %w", err)
		}
	}
	return &Journal{path: clean, streamID: streamID, f: f, entries: entries, repaired: repaired}, nil
}

// syncDirChain fsyncs dir and each newly created ancestor so a power loss
// cannot lose the new journal path.
func syncDirChain(dir string) error {
	if err := syncDir(dir); err != nil {
		return err
	}
	parent := filepath.Dir(dir)
	if parent != dir {
		return syncDir(parent)
	}
	return nil
}

// Read reads and verifies a journal file without creating it or opening it for
// append. It returns (nil, false, nil) when the file does not exist. The
// repaired flag is true when the current file has a torn tail OR a prior repair
// marker is present, so an offline read still reports a past repair.
func Read(path string) ([]Entry, bool, error) {
	clean := filepath.Clean(path)
	entries, repaired, err := readEntries(clean)
	if err != nil {
		return nil, false, err
	}
	// A leftover marker beside a deleted journal describes a journal that no
	// longer exists; journal loss must be reported as loss, not as a repair.
	if _, statErr := os.Stat(clean); statErr == nil && markerPresent(clean) {
		repaired = true
	}
	return entries, repaired, nil
}

// repairMarkerPath is the sidecar file recording that a torn tail was repaired.
func repairMarkerPath(path string) string { return path + ".repair" }

func markerPresent(path string) bool {
	_, err := os.Stat(repairMarkerPath(path))
	return err == nil
}

// writeRepairMarker durably records a torn-tail repair next to the journal.
func writeRepairMarker(path string, discarded int64) error {
	rec := fmt.Sprintf("{\"repaired_at\":%q,\"discarded_bytes\":%d}\n",
		time.Now().UTC().Format(time.RFC3339Nano), discarded)
	marker := repairMarkerPath(path)
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) // #nosec G304 -- operator-selected local journal path
	if err != nil {
		return err
	}
	if _, err := f.WriteString(rec); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return syncDir(filepath.Dir(marker))
}

// degradedMarkerPath is the sidecar file recording that the journal could not
// durably record one or more observations (a write/fsync failure, or a rejected
// oversized entry). Unlike a torn-tail repair, this records observations that
// were observed but never became durable.
func degradedMarkerPath(path string) string { return path + ".degraded" }

// writeDegradedMarker durably records that the journal lost one or more
// observations, so an offline reader can disclose the gap even when the failed
// write left the journal file byte-identical to the last commit.
func writeDegradedMarker(path, reason string) error {
	rec := fmt.Sprintf("{\"degraded_at\":%q,\"reason\":%q}\n",
		time.Now().UTC().Format(time.RFC3339Nano), reason)
	marker := degradedMarkerPath(path)
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) // #nosec G304 -- operator-selected local journal path
	if err != nil {
		return err
	}
	if _, err := f.WriteString(rec); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return syncDir(filepath.Dir(marker))
}

// MarkDegraded durably records that the journal lost an observation. It is
// best-effort: if the marker cannot itself be written, offline disclosure
// cannot be guaranteed (the in-memory journal still refuses further appends
// after a write/fsync failure).
func (j *Journal) MarkDegraded(reason string) error {
	if j == nil {
		return nil
	}
	return writeDegradedMarker(j.path, reason)
}

// Degraded reports whether a durable degradation marker is present beside the
// journal at path, and its recorded reason. It reads only; it never creates.
func Degraded(path string) (bool, string) {
	marker := degradedMarkerPath(filepath.Clean(path))
	data, err := os.ReadFile(marker) // #nosec G304 -- operator-selected local journal path
	if err != nil {
		if os.IsNotExist(err) {
			return false, ""
		}
		// An existing-but-unreadable marker must not be reported as absent;
		// report it degraded rather than fail open.
		return true, "degradation marker unreadable: " + err.Error()
	}
	var rec struct {
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(data, &rec)
	return true, rec.Reason
}

// discardedTailBytes returns the number of bytes in the torn trailing fragment.
func discardedTailBytes(path string) int64 {
	data, err := os.ReadFile(path) // #nosec G304 -- operator-selected local journal path
	if err != nil || len(data) == 0 {
		return 0
	}
	if data[len(data)-1] == '\n' {
		return 0
	}
	return int64(len(data)) - lastCompleteOffset(path)
}

// Path returns the journal file path.
func (j *Journal) Path() string { return j.path }

// StreamID returns the stream identifier bound to this journal.
func (j *Journal) StreamID() string { return j.streamID }

// Repaired reports whether Open repaired a torn trailing line.
func (j *Journal) Repaired() bool { return j.repaired }

// Len returns the number of durable entries.
func (j *Journal) Len() int { return len(j.entries) }

// LastPosition returns the position of the last durable entry, or 0 if empty.
func (j *Journal) LastPosition() int64 {
	if len(j.entries) == 0 {
		return 0
	}
	return j.entries[len(j.entries)-1].Position
}

// Entries returns a copy of all durable entries.
func (j *Journal) Entries() []Entry {
	out := make([]Entry, len(j.entries))
	copy(out, j.entries)
	return out
}

// EntriesFrom returns durable entries with Position greater than position.
func (j *Journal) EntriesFrom(position int64) []Entry {
	out := make([]Entry, 0)
	for _, e := range j.entries {
		if e.Position > position {
			out = append(out, e)
		}
	}
	return out
}

// Append durably writes one observation and returns the stored entry with its
// assigned position and hash. It fsyncs before returning, so a returned entry
// is durable.
func (j *Journal) Append(e Entry) (Entry, error) {
	if j.f == nil {
		return Entry{}, ErrJournalClosed
	}
	if j.failed {
		return Entry{}, fmt.Errorf("%w: journal failed after a prior write error", ErrJournalFailed)
	}
	if e.FormatVersion == 0 {
		e.FormatVersion = JournalFormatVersion
	}
	if e.FormatVersion > JournalFormatVersion {
		return Entry{}, fmt.Errorf("%w: got %d, max %d", ErrJournalVersionUnsupported, e.FormatVersion, JournalFormatVersion)
	}
	if strings.TrimSpace(e.ObservationID) == "" {
		return Entry{}, fmt.Errorf("capturejournal: observation_id required")
	}
	e.Position = j.LastPosition() + 1
	e.PrevHash = hash.GenesisHash
	if n := len(j.entries); n > 0 {
		e.PrevHash = j.entries[n-1].Hash
	}
	h, err := computeHash(e)
	if err != nil {
		return Entry{}, err
	}
	e.Hash = h

	line, err := json.Marshal(e)
	if err != nil {
		return Entry{}, fmt.Errorf("capturejournal: marshal: %w", err)
	}
	line = append(line, '\n')
	if len(line) > maxLineBytes {
		// Reject before writing: a persisted oversized line would poison every
		// subsequent read (the scanner would fail on it). The journal stays
		// usable, but the dropped observation is recorded durably so it cannot
		// be silently lost over a later healthy status.
		_ = writeDegradedMarker(j.path, fmt.Sprintf("rejected oversized entry (%d bytes, limit %d)", len(line), maxLineBytes))
		return Entry{}, fmt.Errorf("%w: entry is %d bytes, limit %d", ErrJournalEntryTooLarge, len(line), maxLineBytes)
	}
	if _, err := j.f.Write(line); err != nil {
		j.failed = true
		// Record the loss durably before returning. A zero-byte write leaves the
		// journal byte-identical to the last commit, so without this marker a
		// restart would report healthy over the dropped observation.
		_ = writeDegradedMarker(j.path, fmt.Sprintf("journal write failed: %v", err))
		return Entry{}, fmt.Errorf("capturejournal: write: %w", err)
	}
	if err := j.f.Sync(); err != nil {
		j.failed = true
		_ = writeDegradedMarker(j.path, fmt.Sprintf("journal fsync failed: %v", err))
		return Entry{}, fmt.Errorf("capturejournal: fsync: %w", err)
	}
	j.entries = append(j.entries, e)
	return e, nil
}

// Sync fsyncs the journal file.
func (j *Journal) Sync() error {
	if j.f == nil {
		return nil
	}
	return j.f.Sync()
}

// Close closes the journal file.
func (j *Journal) Close() error {
	if j.f == nil {
		return nil
	}
	err := j.f.Close()
	j.f = nil
	return err
}

// readEntries parses and verifies every complete line. A non-terminated final
// fragment is reported as a repairable torn tail; any other failure is fatal.
func readEntries(path string) ([]Entry, bool, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- operator-selected local journal path
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("capturejournal: read: %w", err)
	}
	if len(data) == 0 {
		return nil, false, nil
	}

	repaired := false
	if data[len(data)-1] != '\n' {
		// Torn trailing line from a crash mid-append.
		repaired = true
		if idx := lastNewline(data); idx >= 0 {
			data = data[:idx+1]
		} else {
			data = nil
		}
	}

	var entries []Entry
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 64*1024), maxLineBytes)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, false, fmt.Errorf("%w: line %d: %v", ErrJournalCorrupted, lineNo, err)
		}
		if e.FormatVersion != JournalFormatVersion {
			return nil, false, fmt.Errorf("%w: line %d: unsupported format_version %d", ErrJournalVersionUnsupported, lineNo, e.FormatVersion)
		}
		wantPos := int64(len(entries) + 1)
		if e.Position != wantPos {
			return nil, false, fmt.Errorf("%w: line %d: position %d, want %d", ErrJournalCorrupted, lineNo, e.Position, wantPos)
		}
		wantPrev := hash.GenesisHash
		if n := len(entries); n > 0 {
			wantPrev = entries[n-1].Hash
		}
		if e.PrevHash != wantPrev {
			return nil, false, fmt.Errorf("%w: line %d: prev_hash mismatch", ErrJournalCorrupted, lineNo)
		}
		h, err := computeHash(e)
		if err != nil {
			return nil, false, err
		}
		if h != e.Hash {
			return nil, false, fmt.Errorf("%w: line %d: entry hash mismatch", ErrJournalCorrupted, lineNo)
		}
		entries = append(entries, e)
	}
	if err := sc.Err(); err != nil {
		return nil, false, fmt.Errorf("%w: scan: %v", ErrJournalCorrupted, err)
	}
	return entries, repaired, nil
}

func lastNewline(data []byte) int {
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == '\n' {
			return i
		}
	}
	return -1
}

// lastCompleteOffset returns the byte offset of the end of the last complete
// (newline-terminated) line in the file.
func lastCompleteOffset(path string) int64 {
	data, err := os.ReadFile(path) // #nosec G304 -- operator-selected local journal path
	if err != nil || len(data) == 0 {
		return 0
	}
	if data[len(data)-1] == '\n' {
		return int64(len(data))
	}
	if idx := lastNewline(data); idx >= 0 {
		return int64(idx + 1)
	}
	return 0
}
