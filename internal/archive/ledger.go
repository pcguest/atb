// SPDX-License-Identifier: MIT
// Package archive provides a tamper-evident archive ledger for archived ATB bundles.
package archive

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pcguest/atb/internal/hash"
)

const (
	// LedgerFile is the default ledger file name under the archive directory.
	LedgerFile = "index.ndjson"
	// ledgerEventType is the synthetic event type used for hashing ledger entries.
	ledgerEventType = "archive.entry"
)

// Entry is one line in archive.atb/index.ndjson.
type Entry struct {
	Sequence   int    `json:"seq"`
	PrevHash   string `json:"prev_hash"`
	ArchivedAt string `json:"archived_at"`
	Source     string `json:"source"`
	Dest       string `json:"dest"`
	SHA256     string `json:"sha256"`
	HeadHash   string `json:"head_hash"`
}

type ledgerEventData struct {
	ArchivedAt string `json:"archived_at"`
	Source     string `json:"source"`
	Dest       string `json:"dest"`
	SHA256     string `json:"sha256"`
	HeadHash   string `json:"head_hash"`
}

// Load reads all entries from an NDJSON ledger file.
func Load(path string) ([]Entry, error) {
	f, err := os.Open(filepath.Clean(path)) // #nosec G304 -- caller controls local ledger path
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entries := make([]Entry, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("parse archive ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan archive ledger: %w", err)
	}

	return entries, nil
}

// Verify checks sequence continuity and prev_hash linkage and returns the head hash.
func Verify(entries []Entry) (string, error) {
	prev := hash.GenesisHash
	for i, entry := range entries {
		expectedSeq := i + 1
		if entry.Sequence != expectedSeq {
			return "", fmt.Errorf("archive ledger: entry %d has seq %d, expected %d", i, entry.Sequence, expectedSeq)
		}
		if entry.PrevHash != prev {
			return "", fmt.Errorf("archive ledger: entry %d has prev_hash %s, expected %s", i, entry.PrevHash, prev)
		}

		current, err := ComputeHash(entry)
		if err != nil {
			return "", fmt.Errorf("archive ledger: compute hash for seq %d: %w", entry.Sequence, err)
		}
		prev = current
	}
	return prev, nil
}

// ComputeHash calculates the hash for a ledger entry using the shared hash engine.
// The canonicalized payload includes seq and prev_hash via hash.Event.
func ComputeHash(entry Entry) (string, error) {
	e := hash.Event{
		Sequence: entry.Sequence,
		PrevHash: entry.PrevHash,
		Type:     ledgerEventType,
		Data: ledgerEventData{
			ArchivedAt: entry.ArchivedAt,
			Source:     entry.Source,
			Dest:       entry.Dest,
			SHA256:     entry.SHA256,
			HeadHash:   entry.HeadHash,
		},
	}
	return hash.Compute(e)
}

// NextEntry creates the next linked entry from the existing ledger chain.
func NextEntry(existing []Entry, archivedAt, source, dest, fileSHA256, headHash string) (Entry, error) {
	prev, err := Verify(existing)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		Sequence:   len(existing) + 1,
		PrevHash:   prev,
		ArchivedAt: archivedAt,
		Source:     source,
		Dest:       dest,
		SHA256:     fileSHA256,
		HeadHash:   headHash,
	}, nil
}

// Append writes a single NDJSON entry to the ledger file.
func Append(path string, entry Entry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- project-local archive directory
		return fmt.Errorf("archive ledger: mkdir %s: %w", dir, err)
	}
	f, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600) // #nosec G304 -- caller controls local ledger path
	if err != nil {
		return fmt.Errorf("archive ledger: open for append: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(entry); err != nil {
		return fmt.Errorf("archive ledger: append entry: %w", err)
	}
	return nil
}

// LoadOrEmpty loads an existing ledger or returns an empty chain when missing.
func LoadOrEmpty(path string) ([]Entry, error) {
	entries, err := Load(path)
	if err == nil {
		return entries, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return []Entry{}, nil
	}
	return nil, err
}
