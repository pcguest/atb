// SPDX-License-Identifier: MIT
// Package bundle handles reading and writing ATB bundle files (.atb).
// An ATB bundle is a newline-delimited JSON (NDJSON) file where each line
// is a JSON object containing an event and its hash.
package bundle

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	// ManifestVersion is the default manifest format version emitted by writers.
	ManifestVersion = "1"
	// ManifestVersionV2 is the structured-object manifest format.
	ManifestVersionV2 = 2
	// ManifestVersionMax is the highest manifest version this build of atb
	// can read. A bundle whose manifest declares a higher version is
	// rejected with ErrMalformed by parseManifestData so a forward-bumped
	// bundle opened by a stale reader fails loudly instead of silently
	// dropping fields.
	ManifestVersionMax = 2
)

// ErrMalformed, ErrNoManifest, ErrTamper, and ErrNotABundle are declared
// in errors.go alongside the rest of the package's typed sentinel errors.

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

// NewOptions configures bundle construction.
type NewOptions struct {
	// ManifestVersion 0 or 1 uses the stable v1 manifest. 2 uses the
	// experimental structured manifest data object.
	ManifestVersion int
	// CaptureRunID records the capture run that created the bundle. It is
	// omitted when empty and is only intended for bundle-creation provenance.
	CaptureRunID string
}

// ManifestData is the structured payload of a manifest event.
type ManifestData struct {
	Version      string `json:"version"`
	CreatedAt    string `json:"created_at"` // RFC 3339 UTC
	BundleID     string `json:"bundle_id"`  // random 16-byte hex
	CaptureRunID string `json:"capture_run_id,omitempty"`
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
	return NewWithOptions(NewOptions{})
}

// NewWithOptions creates a new empty bundle with explicit construction options.
func NewWithOptions(opts NewOptions) (*Bundle, error) {
	b := &Bundle{}
	createdAt := time.Now().UTC().Format(time.RFC3339)
	bundleID, err := newBundleID()
	if err != nil {
		return nil, err
	}

	switch opts.ManifestVersion {
	case 0, 1:
	case ManifestVersionV2:
		data := map[string]any{
			"version":    ManifestVersionV2,
			"created_at": createdAt,
			"bundle_id":  bundleID,
		}
		if opts.CaptureRunID != "" {
			data["capture_run_id"] = opts.CaptureRunID
		}
		if err := b.AppendWithOptions(ManifestEventType, data, &AppendOptions{
			Timestamp: createdAt,
		}); err != nil {
			return nil, fmt.Errorf("bundle: new: append manifest: %w", err)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("bundle: new: unsupported manifest version %d", opts.ManifestVersion)
	}

	// Manifest v1 intentionally stores the structured manifest fields as a
	// JSON-encoded string in event.data. Readers must decode data a second
	// time; changing this requires a manifest version bump.
	manifestPayload, err := json.Marshal(ManifestData{
		Version:      ManifestVersion,
		CreatedAt:    createdAt,
		BundleID:     bundleID,
		CaptureRunID: opts.CaptureRunID,
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
	if eventType != ManifestEventType && hasManifestRecord(b.Records) {
		if _, err := parseManifestData(&b.Records[0]); err != nil {
			return fmt.Errorf("bundle: append with options: parse manifest: %w", err)
		}
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
	// Legacy: pre-v1.0 bundles without a manifest record. Do not remove until
	// the migration policy in VERSIONING.md declares them unsupported. The
	// fallthrough above (sequence = len(b.Records) + 1) is the legacy path.

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
// It writes to a temp file in the same directory first and then renames
// atomically so a crash mid-write cannot truncate the existing bundle.
//
// Save acquires an exclusive advisory lock on `path + ".lock"` for the
// duration of the write. Concurrent callers that find the lock held
// receive an error matchable via errors.Is(err, ErrBundleLocked); they
// do not block. See internal/bundle/lock_unix.go for the contract.
func (b *Bundle) Save(args ...any) error {
	ctx, path, err := contextPathArgs("bundle: save", args)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err = withBundleLock(path, func() error {
		return b.saveUnlocked(ctx, path)
	})
	if errors.Is(err, ErrBundleLocked) {
		return fmt.Errorf("bundle: save: %w", err)
	}
	return err
}

// SaveWithRetry writes the bundle after acquiring the advisory lock, retrying
// lock contention until timeout elapses or ctx is cancelled. A zero timeout
// preserves Save's fail-fast behaviour.
func (b *Bundle) SaveWithRetry(ctx context.Context, path string, timeout, interval time.Duration) error {
	return b.saveWithLock(ctx, path, func(path string) (func() error, error) {
		return AcquireWithRetry(ctx, path, timeout, interval)
	})
}

func (b *Bundle) saveWithLock(ctx context.Context, path string, acquire func(string) (func() error, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	release, err := acquire(path)
	if err != nil {
		return fmt.Errorf("bundle: save: %w", err)
	}
	defer func() {
		_ = release()
	}()

	return b.saveUnlocked(ctx, path)
}

func (b *Bundle) saveUnlocked(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- tightened to 0750 per gosec
		return fmt.Errorf("bundle: save: mkdir: %w", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, r := range b.Records {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("bundle: save: encode: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if err := writeAtomic(path, buf.Bytes(), bundleFileMode); err != nil {
		return fmt.Errorf("bundle: save: %w", err)
	}
	return nil
}

// Load parses path as NDJSON into a Bundle without verifying the hash chain.
// Use LoadVerified for safe loading. This function is intentionally non-validating
// for tooling that needs to inspect malformed or partial bundles.
func Load(args ...any) (*Bundle, error) {
	ctx, path, err := contextPathArgs("bundle: load", args)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Clean(path)) // #nosec G304 -- path is user-specified for CLI; caller validates
	if err != nil {
		return nil, fmt.Errorf("bundle: load: open: %w", err)
	}
	defer f.Close()
	b, err := LoadReader(f)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return b, nil
}

// LoadVerified loads the bundle at path, confirms record 0 is the manifest,
// and verifies the hash chain. Returns a typed error for every failure class:
//   - ErrNotABundle  if record 0 is not the manifest type
//   - ErrNoManifest  if the bundle has no records at all
//   - ErrTamper      (via Verify) if the chain is broken
//   - ErrMalformed   (via Load) if the NDJSON cannot be parsed
//
// For non-bundle-format errors (file not found, permission denied), the
// underlying os error is returned unwrapped.
func LoadVerified(path string) (*Bundle, error) {
	b, err := Load(context.Background(), path)
	if err != nil {
		return nil, err
	}
	if len(b.Records) == 0 {
		return nil, fmt.Errorf("empty file: %w", ErrNoManifest)
	}
	if b.Records[0].Event.Type != ManifestEventType {
		return nil, fmt.Errorf("record 0 type %q: %w", b.Records[0].Event.Type, ErrNotABundle)
	}
	if err := b.Verify(); err != nil {
		return nil, err
	}
	return b, nil
}

func contextPathArgs(op string, args []any) (context.Context, string, error) {
	var (
		ctx  context.Context
		path string
	)
	switch len(args) {
	case 1:
		ctx = context.Background()
		var ok bool
		path, ok = args[0].(string)
		if !ok {
			return nil, "", fmt.Errorf("%s: expected path string", op)
		}
	case 2:
		var ok bool
		ctx, ok = args[0].(context.Context)
		if !ok {
			return nil, "", fmt.Errorf("%s: expected context.Context", op)
		}
		path, ok = args[1].(string)
		if !ok {
			return nil, "", fmt.Errorf("%s: expected path string", op)
		}
	default:
		return nil, "", fmt.Errorf("%s: expected path or context and path", op)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx, path, nil
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
			return nil, fmt.Errorf("bundle: load: unmarshal: %w", errors.Join(err, ErrMalformed))
		}
		b.Records = append(b.Records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("bundle: load: scan: %w", err)
	}
	if hasManifestRecord(b.Records) {
		if _, err := parseManifestData(&b.Records[0]); err != nil {
			return nil, fmt.Errorf("bundle: load: parse manifest: %w", err)
		}
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
//
// ManifestE is preferred; Manifest swallows errors and returns nil on failure.
func (b *Bundle) Manifest() *ManifestData {
	if !hasManifestRecord(b.Records) {
		return nil
	}

	manifest, err := parseManifestData(&b.Records[0])
	if err != nil {
		return nil
	}
	return &manifest
}

// ManifestE returns the manifest data and a typed error.
// Returns (nil, ErrNoManifest) if the bundle has no records or the first
// record is not a manifest type. Returns (nil, err) wrapping ErrMalformed
// if the manifest data is present but cannot be decoded.
// Returns (*ManifestData, nil) on success.
func (b *Bundle) ManifestE() (*ManifestData, error) {
	if !hasManifestRecord(b.Records) {
		return nil, ErrNoManifest
	}
	manifest, err := parseManifestData(&b.Records[0])
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

func parseManifestData(record *Record) (ManifestData, error) {
	if record == nil || record.Event.Type != ManifestEventType {
		return ManifestData{}, fmt.Errorf("bundle: manifest record absent")
	}

	version, err := manifestVersionFromRecord(record)
	if err != nil {
		return ManifestData{}, err
	}
	if version > ManifestVersionMax {
		return ManifestData{}, fmt.Errorf(
			"bundle: unsupported manifest version %d: this bundle requires a newer version of atb to open: %w",
			version, ErrMalformed,
		)
	}

	switch data := record.Event.Data.(type) {
	case string:
		// v1 (or absent — legacy bundles pre-versioned manifest) — string-encoded JSON path.
		return parseManifestDataV1(data)
	case map[string]any:
		// v2 — structured-object path.
		return parseManifestDataV2(data)
	default:
		return ManifestData{}, fmt.Errorf("bundle: manifest data type %T is not supported: %w", record.Event.Data, ErrMalformed)
	}
}

// manifestVersionFromRecord extracts the integer version from a manifest
// record without committing to either parser path. It returns 1 when the
// version field is absent (legacy bundles pre-versioned manifest).
func manifestVersionFromRecord(record *Record) (int, error) {
	switch data := record.Event.Data.(type) {
	case string:
		var probe struct {
			Version any `json:"version"`
		}
		if err := json.Unmarshal([]byte(data), &probe); err != nil {
			return 0, fmt.Errorf("bundle: manifest data not valid JSON: %w", errors.Join(err, ErrMalformed))
		}
		return manifestVersionToInt(probe.Version)
	case map[string]any:
		return manifestVersionToInt(data["version"])
	default:
		return 0, fmt.Errorf("bundle: manifest data type %T is not supported: %w", record.Event.Data, ErrMalformed)
	}
}

// manifestVersionToInt normalises the JSON-decoded version field across the
// shapes encountered in the wild: float64 (from json.Unmarshal of a number),
// int (from in-memory maps), and string. Absent or empty version is treated
// as v1 for backward compatibility with legacy bundles.
func manifestVersionToInt(value any) (int, error) {
	switch v := value.(type) {
	case nil:
		return 1, nil
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		if v == "" {
			return 1, nil
		}
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
			return 0, fmt.Errorf("bundle: manifest version %q is not an integer: %w", v, ErrMalformed)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("bundle: manifest version %v has unsupported type %T: %w", value, value, ErrMalformed)
	}
}

func parseManifestDataV1(raw string) (ManifestData, error) {
	var manifest ManifestData
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		return ManifestData{}, err
	}
	if manifest.Version == "2" {
		return ManifestData{}, fmt.Errorf("bundle: v2 manifest is structured and cannot be parsed by the v1 path")
	}
	return manifest, nil
}

func parseManifestDataV2(fields map[string]any) (ManifestData, error) {
	version, err := manifestVersionString(fields["version"])
	if err != nil {
		return ManifestData{}, err
	}
	createdAt, _ := fields["created_at"].(string)
	bundleID, _ := fields["bundle_id"].(string)
	captureRunID, _ := fields["capture_run_id"].(string)
	if version != "2" {
		return ManifestData{}, fmt.Errorf("bundle: structured manifest version %s is not supported", version)
	}
	if createdAt == "" || bundleID == "" {
		return ManifestData{}, fmt.Errorf("bundle: structured manifest missing created_at or bundle_id")
	}
	return ManifestData{
		Version:      version,
		CreatedAt:    createdAt,
		BundleID:     bundleID,
		CaptureRunID: captureRunID,
	}, nil
}

func manifestVersionString(value any) (string, error) {
	switch v := value.(type) {
	case float64:
		if v == ManifestVersionV2 {
			return "2", nil
		}
	case int:
		if v == ManifestVersionV2 {
			return "2", nil
		}
	case string:
		if v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("bundle: manifest version %v is not supported", value)
}

func newBundleID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("bundle: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hasManifestRecord(records []Record) bool {
	return len(records) > 0 && records[0].Event.Type == ManifestEventType
}

func verifyManifestBundle(records []Record) error {
	prev := hash.GenesisHash
	for i, record := range records {
		expectedSeq := i

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
				"bundle: verify: sequence mismatch at index %d: expected seq %d, got seq %d: %w",
				i,
				expectedSeq,
				record.Event.Sequence,
				ErrTamper,
			)
		}
		if storedPrevHash != prev {
			return fmt.Errorf(
				"bundle: verify: prev_hash mismatch at index %d: expected %s, got %s: %w",
				i,
				prev,
				storedPrevHash,
				ErrTamper,
			)
		}
		if computed != record.Hash {
			return fmt.Errorf(
				"bundle: verify: tamper detected at event %d (seq %d): expected %s, got %s: %w",
				i,
				expectedSeq,
				record.Hash,
				computed,
				ErrTamper,
			)
		}
		prev = computed
	}
	return nil
}
