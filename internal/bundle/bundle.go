// Package bundle handles reading and writing ATB bundle files (.atb).
// An ATB bundle is a newline-delimited JSON (NDJSON) file where each line
// is a JSON object containing an event and its hash.
package bundle

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pcguest/atb/internal/hash"
)

const (
	// BundleDir is the default directory for ATB bundles.
	BundleDir = "run.atb"
	// BundleFile is the default bundle filename.
	BundleFile = "bundle.atb"
)

// Record is a single line in an ATB bundle file.
type Record struct {
	Event hash.Event `json:"event"`
	Hash  string     `json:"hash"`
}

// Bundle represents an in-memory ATB bundle.
type Bundle struct {
	Records []Record
}

// AppendOptions carries optional identity metadata for multi-tenant event contexts.
type AppendOptions struct {
	ActorID     *string
	OrgID       *string
	WorkspaceID *string
}

// New creates a new empty bundle.
func New() *Bundle {
	return &Bundle{}
}

// Append adds a new event to the bundle, computing its hash automatically.
func (b *Bundle) Append(eventType string, data interface{}) error {
	return b.AppendWithOptions(eventType, data, nil)
}

// AppendWithOptions adds a new event with optional identity metadata.
func (b *Bundle) AppendWithOptions(eventType string, data interface{}, opts *AppendOptions) error {
	prevHash := hash.GenesisHash
	if len(b.Records) > 0 {
		prevHash = b.Records[len(b.Records)-1].Hash
	}
	e := hash.Event{
		Sequence: len(b.Records) + 1,
		PrevHash: prevHash,
		Type:     eventType,
		Data:     data,
	}
	if opts != nil {
		e.ActorID = opts.ActorID
		e.OrgID = opts.OrgID
		e.WorkspaceID = opts.WorkspaceID
	}
	h, err := hash.Compute(e)
	if err != nil {
		return fmt.Errorf("bundle: append with options: %w", err)
	}
	b.Records = append(b.Records, Record{Event: e, Hash: h})
	return nil
}

// Verify checks the integrity of the entire bundle.
func (b *Bundle) Verify() error {
	events := make([]hash.Event, len(b.Records))
	hashes := make([]string, len(b.Records))
	for i, r := range b.Records {
		events[i] = r.Event
		hashes[i] = r.Hash
	}
	return hash.Verify(events, hashes)
}

// Save writes the bundle to the given file path in NDJSON format.
func (b *Bundle) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil { // #nosec G301 -- tightened to 0750 per gosec
		return fmt.Errorf("bundle: save: mkdir: %w", err)
	}
	f, err := os.Create(filepath.Clean(path)) // #nosec G304 -- path is user-specified for CLI; caller validates
	if err != nil {
		return fmt.Errorf("bundle: save: create: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range b.Records {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("bundle: save: encode: %w", err)
		}
	}
	return nil
}

// Load reads a bundle from the given file path.
func Load(path string) (*Bundle, error) {
	f, err := os.Open(filepath.Clean(path)) // #nosec G304 -- path is user-specified for CLI; caller validates
	if err != nil {
		return nil, fmt.Errorf("bundle: load: open: %w", err)
	}
	defer f.Close()
	b := New()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("bundle: load: unmarshal: %w", err)
		}
		b.Records = append(b.Records, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("bundle: load: scan: %w", err)
	}
	return b, nil
}

// DefaultPath returns the default bundle path relative to the current directory.
func DefaultPath() string {
	return filepath.Join(BundleDir, BundleFile)
}
