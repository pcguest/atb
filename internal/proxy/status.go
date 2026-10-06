// SPDX-License-Identifier: MIT
package proxy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pcguest/atb/internal/acquisition"
	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/capturejournal"
)

// CaptureStatus is the offline operator state of an intercept capture run.
//
// It deliberately separates process liveness from capture continuity: a status
// read cannot observe a live process, so process_health is always "unknown"
// here, and a healthy-looking bundle must never be read as complete capture.
type CaptureStatus struct {
	BundlePath    string `json:"bundle_path"`
	Integrity     string `json:"integrity"`
	ProcessHealth string `json:"process_health"`
	CaptureState  string `json:"capture_state"`

	SourceSystem      string `json:"source_system,omitempty"`
	SourceIncarnation string `json:"source_incarnation,omitempty"`
	Adapter           string `json:"adapter,omitempty"`
	AdapterVersion    string `json:"adapter_version,omitempty"`

	JournalPath         string `json:"journal_path,omitempty"`
	JournalEntries      int    `json:"journal_entries"`
	JournalLastPosition int64  `json:"journal_last_position"`
	CommittedPosition   int64  `json:"committed_position"`
	JournalBacklog      int64  `json:"journal_backlog"`

	LastObservationAt   string `json:"last_observation_at,omitempty"`
	LastDurableCommitAt string `json:"last_durable_commit_at,omitempty"`

	KnownGap           bool   `json:"known_gap"`
	PossibleUnknownGap bool   `json:"possible_unknown_gap"`
	Detail             string `json:"detail,omitempty"`
}

// ReadCaptureStatus reads the persisted capture state for a bundle offline. It
// never writes and never creates the journal.
func ReadCaptureStatus(bundlePath string) (*CaptureStatus, error) {
	clean := filepath.Clean(bundlePath)
	streamID := filepath.Base(clean)
	journalPath := filepath.Join(filepath.Dir(clean), captureJournalDir, streamID+".journal.ndjson")
	checkpointPath := acquisition.ResolveCheckpointPath(clean, captureSourceSystem, streamID)

	st := &CaptureStatus{
		BundlePath:    clean,
		Integrity:     "unknown",
		ProcessHealth: "unknown",
		CaptureState:  string(captureUnknown),
		JournalPath:   journalPath,
	}

	b, err := bundle.LoadVerified(clean)
	if err != nil {
		if os.IsNotExist(err) {
			st.Detail = "bundle not found"
			return st, nil
		}
		st.Integrity = "failed"
		st.CaptureState = string(captureDegraded)
		st.Detail = "bundle failed integrity verification"
		return st, nil
	}
	st.Integrity = "verified"

	hasAcquisition := false
	for _, rec := range b.Records {
		if rec.Event.Acquisition != nil {
			hasAcquisition = true
			break
		}
	}

	cp, cpErr := acquisition.Load(checkpointPath)
	checkpointPresent := cpErr == nil
	if cpErr != nil && !errors.Is(cpErr, acquisition.ErrCheckpointNotFound) {
		st.CaptureState = string(captureDegraded)
		st.Detail = fmt.Sprintf("checkpoint unreadable: %v", cpErr)
		return st, nil
	}
	if checkpointPresent {
		st.SourceSystem = cp.SourceSystem
		st.SourceIncarnation = cp.SourceIncarnation
		st.Adapter = cp.Adapter
		st.AdapterVersion = cp.AdapterVersion
		st.CommittedPosition = parsePosition(cp.Position)
		st.LastDurableCommitAt = cp.ObservedAt
	}

	entries, repaired, err := capturejournal.Read(journalPath)
	if err != nil {
		st.CaptureState = string(captureDegraded)
		st.KnownGap = true
		st.Detail = "journal corrupted; continuity cannot be established"
		return st, nil
	}
	st.JournalEntries = len(entries)
	if n := len(entries); n > 0 {
		st.JournalLastPosition = entries[n-1].Position
		st.LastObservationAt = entries[n-1].ObservedAt
	}
	if st.JournalLastPosition > st.CommittedPosition {
		st.JournalBacklog = st.JournalLastPosition - st.CommittedPosition
	}
	st.KnownGap = repaired

	// Continuity assessment.
	switch {
	case len(entries) == 0 && !checkpointPresent && !hasAcquisition:
		st.CaptureState = "not_established"
		st.Detail = "no capture journal, checkpoint, or acquisition evidence present"
	case repaired:
		st.CaptureState = string(captureDegraded)
		st.Detail = "journal torn tail repaired on read; observations may be incomplete"
	case st.JournalBacklog > 0:
		st.CaptureState = string(captureRecoveryRequired)
		st.Detail = "durable observations exist that are not yet committed to evidence"
	case checkpointPresent && cp.BundleHeadHash != "":
		if err := validateCheckpointPrefix(cp, b); err != nil {
			st.CaptureState = string(captureDegraded)
			st.Detail = "checkpoint binding does not match the evidence"
			break
		}
		st.CaptureState = string(captureHealthy)
	case hasAcquisition:
		st.CaptureState = string(captureUnknown)
		st.PossibleUnknownGap = true
		st.Detail = "acquisition evidence present but no checkpoint to establish continuity"
	default:
		st.CaptureState = string(captureUnknown)
	}
	return st, nil
}
