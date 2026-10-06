// SPDX-License-Identifier: MIT
package proxy

import (
	"errors"
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

	v3 := false
	hasAcquisition := false
	for _, rec := range b.Records {
		if rec.Event.Acquisition != nil {
			hasAcquisition = true
			break
		}
	}
	if m := b.Manifest(); m != nil && m.Version == "3" {
		v3 = true
	}

	// Read journal state independently of checkpoint readability, so operators
	// can still see durable recovery material when the checkpoint is damaged.
	entries, repaired, jerr := capturejournal.Read(journalPath)
	if jerr != nil {
		st.KnownGap = true
	}
	st.JournalEntries = len(entries)
	if n := len(entries); n > 0 {
		st.JournalLastPosition = entries[n-1].Position
		st.LastObservationAt = entries[n-1].ObservedAt
	}

	cp, cpErr := acquisition.Load(checkpointPath)
	checkpointPresent := cpErr == nil
	checkpointUnreadable := cpErr != nil && !errors.Is(cpErr, acquisition.ErrCheckpointNotFound)
	if checkpointPresent {
		st.SourceSystem = cp.SourceSystem
		st.SourceIncarnation = cp.SourceIncarnation
		st.Adapter = cp.Adapter
		st.AdapterVersion = cp.AdapterVersion
		if pos, err := checkpointPosition(cp); err == nil {
			st.CommittedPosition = pos
		} else {
			checkpointUnreadable = true
		}
		st.LastDurableCommitAt = cp.ObservedAt
	}
	if st.JournalLastPosition > st.CommittedPosition {
		st.JournalBacklog = st.JournalLastPosition - st.CommittedPosition
	}

	// Continuity assessment. A healthy state requires a verified v3 bundle, a
	// readable checkpoint bound to this capture's source/adapter/version, and a
	// journal that is not behind the committed position.
	switch {
	case !v3:
		st.CaptureState = string(captureDegraded)
		st.Detail = "bundle is not manifest v3; continuous capture does not apply"
		return st, nil
	case len(entries) == 0 && !checkpointPresent && !hasAcquisition:
		st.CaptureState = "not_established"
		st.Detail = "no capture journal, checkpoint, or acquisition evidence present"
	case jerr != nil:
		st.CaptureState = string(captureDegraded)
		st.KnownGap = true
		st.Detail = "journal corrupted; continuity cannot be established"
	case repaired:
		st.CaptureState = string(captureDegraded)
		st.KnownGap = true
		st.Detail = "journal torn tail detected (repaired on open); observations may be incomplete"
	case checkpointUnreadable:
		st.CaptureState = string(captureDegraded)
		st.Detail = "checkpoint unreadable or malformed; continuity cannot be established"
	case st.JournalLastPosition < st.CommittedPosition:
		st.CaptureState = string(captureDegraded)
		st.PossibleUnknownGap = true
		st.Detail = "journal ends before the committed position; observations may be missing"
	case st.JournalBacklog > 0:
		st.CaptureState = string(captureRecoveryRequired)
		st.Detail = "durable observations exist that are not yet committed to evidence"
	case checkpointPresent:
		if err := cp.Validate(captureSourceSystem, streamID, captureAdapter, captureAdapterVersion, cp.SourceIncarnation); err != nil {
			st.CaptureState = string(captureDegraded)
			st.Detail = "checkpoint does not match this capture's source/adapter/version"
			break
		}
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
