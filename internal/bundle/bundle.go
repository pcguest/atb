// Package bundle handles reading and writing ATB bundle files (.atb).
// An ATB bundle is a newline-delimited JSON (NDJSON) file where each line
// is a JSON object containing an event and its hash.
package bundle

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pcguest/atb/internal/hash"
)

const (
	// BundleDir is the default directory for ATB bundles.
	BundleDir = "run.atb"
	// BundleFile is the default bundle filename.
	BundleFile = "bundle.atb"
	// MaxLineSizeBytes is the maximum supported size of a single NDJSON record.
	MaxLineSizeBytes = 16 * 1024 * 1024
	// ManifestEventType is the reserved event type for bundle manifest records.
	// A manifest record is always seq 0 and is the first record in a new bundle.
	ManifestEventType = "atb.bundle.manifest"
	// AnchorEventType is the reserved event type for TSA anchor records.
	AnchorEventType = "atb.bundle.anchor"
	// ManifestVersion is the current manifest format version.
	ManifestVersion = "1"
)

var eventTypeRegexp = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// Record is a single line in an ATB bundle file.
type Record struct {
	Event hash.Event `json:"event"`
	Hash  string     `json:"hash"`
}

// Bundle represents an in-memory ATB bundle.
type Bundle struct {
	Records []Record
}

// ManifestData is the structured payload of a manifest event.
type ManifestData struct {
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"` // RFC 3339 UTC
	BundleID  string `json:"bundle_id"`  // random 16-byte hex
}

// AppendOptions carries optional identity metadata for multi-tenant event contexts.
type AppendOptions struct {
	ActorID      *string
	OrgID        *string
	WorkspaceID  *string
	Timestamp    string
	TraceID      string
	SpanID       string
	ParentSpanID string
}

// New creates a new empty bundle.
func New() (*Bundle, error) {
	b := &Bundle{}
	createdAt := time.Now().UTC().Format(time.RFC3339)
	manifestPayload, err := json.Marshal(ManifestData{
		Version:   ManifestVersion,
		CreatedAt: createdAt,
		BundleID:  newBundleID(),
	})
	if err != nil {
		return nil, fmt.Errorf("bundle: new: marshal manifest: %w", err)
	}
	if err := b.AppendWithOptions(ManifestEventType, string(manifestPayload), &AppendOptions{
		Timestamp: createdAt,
	}); err != nil {
		return nil, fmt.Errorf("bundle: new: append manifest: %w", err)
	}
	return b, nil
}

// Append adds a new event to the bundle, computing its hash automatically.
func (b *Bundle) Append(eventType string, data interface{}) error {
	return b.AppendWithOptions(eventType, data, nil)
}

// AppendWithOptions adds a new event with optional identity metadata.
func (b *Bundle) AppendWithOptions(eventType string, data interface{}, opts *AppendOptions) error {
	if eventType == ManifestEventType && len(b.Records) > 0 {
		return fmt.Errorf("bundle: manifest record must be the first record in a new bundle")
	}

	prevHash := hash.GenesisHash
	if len(b.Records) > 0 {
		prevHash = b.Records[len(b.Records)-1].Hash
	}
	if !eventTypeRegexp.MatchString(eventType) {
		return fmt.Errorf("bundle: event type %q does not match required pattern (e.g. \"ai.tool.exec\")", eventType)
	}

	sequence := len(b.Records) + 1
	if eventType == ManifestEventType {
		sequence = 0
	} else if hasManifestRecord(b.Records) {
		sequence = len(b.Records)
	}

	e := hash.Event{
		Sequence: sequence,
		PrevHash: prevHash,
		Type:     eventType,
		HashAlgo: "sha256",
		Data:     data,
	}
	if opts != nil {
		e.ActorID = opts.ActorID
		e.OrgID = opts.OrgID
		e.WorkspaceID = opts.WorkspaceID
		if opts.Timestamp != "" {
			e.Timestamp = opts.Timestamp
		}
		if opts.TraceID != "" {
			e.TraceID = opts.TraceID
		}
		if opts.SpanID != "" {
			e.SpanID = opts.SpanID
		}
		if opts.ParentSpanID != "" {
			e.ParentSpanID = opts.ParentSpanID
		}
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
	if hasManifestRecord(b.Records) {
		return verifyManifestBundle(b.Records)
	}

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
	return LoadReader(f)
}

// LoadReader reads a bundle from r in NDJSON format.
// It is the streaming counterpart of Load and is used by atb verify --remote
// to verify a bundle downloaded from S3 without writing a temporary file.
func LoadReader(r io.Reader) (*Bundle, error) {
	b := &Bundle{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), MaxLineSizeBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("bundle: load: unmarshal: %w", err)
		}
		b.Records = append(b.Records, rec)
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

// Manifest returns the parsed manifest data from the first record, if present.
// Returns nil if the bundle has no manifest record or if the first record is not
// of type ManifestEventType.
func (b *Bundle) Manifest() *ManifestData {
	if !hasManifestRecord(b.Records) {
		return nil
	}

	raw, ok := b.Records[0].Event.Data.(string)
	if !ok {
		return nil
	}

	var manifest ManifestData
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		return nil
	}
	return &manifest
}

func newBundleID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hasManifestRecord(records []Record) bool {
	return len(records) > 0 && records[0].Event.Type == ManifestEventType
}

func verifyManifestBundle(records []Record) error {
	prev := hash.GenesisHash
	for i, record := range records {
		expectedSeq := i
		if i == 0 {
			expectedSeq = 0
		}

		event := record.Event
		storedPrevHash := record.Event.PrevHash
		event.PrevHash = prev
		event.Sequence = expectedSeq

		computed, err := hash.Compute(event)
		if err != nil {
			return fmt.Errorf("bundle: verify at index %d: %w", i, err)
		}
		if record.Event.Sequence != expectedSeq {
			return fmt.Errorf(
				"bundle: verify: sequence mismatch at index %d: expected seq %d, got seq %d",
				i,
				expectedSeq,
				record.Event.Sequence,
			)
		}
		if storedPrevHash != prev {
			return fmt.Errorf(
				"bundle: verify: prev_hash mismatch at index %d: expected %s, got %s",
				i,
				prev,
				storedPrevHash,
			)
		}
		if computed != record.Hash {
			return fmt.Errorf(
				"bundle: verify: tamper detected at event %d (seq %d): expected %s, got %s",
				i,
				expectedSeq,
				record.Hash,
				computed,
			)
		}
		prev = computed
	}
	return nil
}
