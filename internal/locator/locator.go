// SPDX-License-Identifier: MIT
// Package locator implements ATB semantic evidence locators: a versioned,
// canonical reference to a single evidence record inside an ATB bundle.
//
// A locator identifies evidence. It does not grant authority, establish
// factual truth, or imply governance approval, and resolving a locator is
// not an integrity verification of the referenced evidence.
//
// Canonical grammar (grammar version 1):
//
//	atb://evidence/1/<bundle_head_hash>?seq=<event_sequence>[&record=<record_hash>]
//
// The grammar is a reference layer over existing evidence identity. It does
// not change canonicalisation, the hash chain, the bundle format, or any
// event field.
package locator

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
)

// GrammarVersion is the locator grammar version supported by this build.
const GrammarVersion = 1

const (
	scheme    = "atb"
	authority = "evidence"
)

// Sentinel errors. Callers should treat these with errors.Is.
//
// A failure to locate evidence is not evidence that the record is invalid,
// and a successful location is not proof that the evidence is trustworthy,
// complete, or factually correct. Location, integrity, and interpretation
// are separate questions.
var (
	// ErrMalformed is returned when a locator is syntactically invalid.
	ErrMalformed = errors.New("locator: malformed")
	// ErrUnsupportedVersion is returned when a locator declares a grammar
	// version this build does not understand. Unknown versions fail closed
	// rather than being guessed.
	ErrUnsupportedVersion = errors.New("locator: unsupported version")
	// ErrBundleUnavailable is returned when the locator cannot be resolved
	// against the supplied bundle (for example a head-hash mismatch, or no
	// bundle at all).
	ErrBundleUnavailable = errors.New("locator: bundle unavailable")
	// ErrEventNotFound is returned when the locator's event sequence is not
	// present in the resolved bundle.
	ErrEventNotFound = errors.New("locator: event not found")
	// ErrRecordHashMismatch is returned when an optional record hash is
	// present and does not match the referenced record's stored hash.
	ErrRecordHashMismatch = errors.New("locator: record hash mismatch")
)

var (
	hex64Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	seqPattern   = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
)

// Locator is a parsed semantic reference to one evidence record.
//
// BundleHeadHash is the bundle's terminal record hash, EventSequence is the
// value of the target record's `seq` field (0 for the manifest record in
// bundles that have one; 1-based positions in legacy manifest-less bundles),
// and RecordHash is an optional integrity cross-check.
type Locator struct {
	BundleHeadHash string
	EventSequence  int
	RecordHash     string
}

// Parse parses and validates a locator string, returning a Locator whose
// String method emits the canonical serialised form.
//
// Parsing is strict: percent-encoding, fragments, unknown query parameters,
// duplicate parameters, and non-canonical hashes are rejected so that a
// locator has exactly one canonical representation.
func Parse(raw string) (Locator, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Locator{}, fmt.Errorf("%w: empty locator", ErrMalformed)
	}
	if strings.ContainsRune(s, '%') {
		return Locator{}, fmt.Errorf("%w: percent-encoding is not permitted", ErrMalformed)
	}

	u, err := url.Parse(s)
	if err != nil {
		return Locator{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if u.Scheme != scheme {
		return Locator{}, fmt.Errorf("%w: scheme %q is not %q", ErrMalformed, u.Scheme, scheme)
	}
	if u.Host != authority {
		return Locator{}, fmt.Errorf("%w: authority %q is not %q", ErrMalformed, u.Host, authority)
	}
	if u.User != nil || u.Port() != "" {
		return Locator{}, fmt.Errorf("%w: userinfo and port are not permitted", ErrMalformed)
	}
	if u.Fragment != "" {
		return Locator{}, fmt.Errorf("%w: fragment is not permitted", ErrMalformed)
	}

	segments := strings.Split(u.Path, "/")
	if len(segments) != 3 || segments[0] != "" {
		return Locator{}, fmt.Errorf("%w: path must be /<version>/<bundle_head_hash>", ErrMalformed)
	}
	version, err := strconv.Atoi(segments[1])
	if err != nil {
		return Locator{}, fmt.Errorf("%w: version segment %q is not an integer", ErrMalformed, segments[1])
	}
	if version != GrammarVersion {
		return Locator{}, fmt.Errorf("%w: locator grammar version %d is not supported", ErrUnsupportedVersion, version)
	}
	head := segments[2]
	if !hex64Pattern.MatchString(head) {
		return Locator{}, fmt.Errorf("%w: bundle head hash must be 64 lowercase hex characters", ErrMalformed)
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return Locator{}, fmt.Errorf("%w: query: %v", ErrMalformed, err)
	}
	for key := range query {
		if key != "seq" && key != "record" {
			return Locator{}, fmt.Errorf("%w: unknown parameter %q", ErrMalformed, key)
		}
	}
	seqValues, ok := query["seq"]
	if !ok || len(seqValues) != 1 {
		return Locator{}, fmt.Errorf("%w: exactly one seq parameter is required", ErrMalformed)
	}
	if !seqPattern.MatchString(seqValues[0]) {
		return Locator{}, fmt.Errorf("%w: seq must be a non-negative decimal integer", ErrMalformed)
	}
	seq, err := strconv.Atoi(seqValues[0])
	if err != nil {
		return Locator{}, fmt.Errorf("%w: seq is out of range", ErrMalformed)
	}

	record := ""
	if values, ok := query["record"]; ok {
		if len(values) != 1 {
			return Locator{}, fmt.Errorf("%w: duplicate record parameter", ErrMalformed)
		}
		if !hex64Pattern.MatchString(values[0]) {
			return Locator{}, fmt.Errorf("%w: record hash must be 64 lowercase hex characters", ErrMalformed)
		}
		record = values[0]
	}

	loc := Locator{BundleHeadHash: head, EventSequence: seq, RecordHash: record}
	if err := loc.Validate(); err != nil {
		return Locator{}, err
	}
	return loc, nil
}

// Validate checks a Locator's field-level invariants.
func (l Locator) Validate() error {
	if !hex64Pattern.MatchString(l.BundleHeadHash) {
		return fmt.Errorf("%w: bundle head hash must be 64 lowercase hex characters", ErrMalformed)
	}
	if l.EventSequence < 0 {
		return fmt.Errorf("%w: event sequence must be a non-negative integer", ErrMalformed)
	}
	if l.RecordHash != "" && !hex64Pattern.MatchString(l.RecordHash) {
		return fmt.Errorf("%w: record hash must be 64 lowercase hex characters", ErrMalformed)
	}
	return nil
}

// String returns the canonical serialised locator. It is safe to call on a
// zero Locator; callers should Validate before treating the result as a
// meaningful reference.
func (l Locator) String() string {
	var b strings.Builder
	b.WriteString(scheme)
	b.WriteString("://")
	b.WriteString(authority)
	b.WriteString("/")
	b.WriteString(strconv.Itoa(GrammarVersion))
	b.WriteString("/")
	b.WriteString(l.BundleHeadHash)
	b.WriteString("?seq=")
	b.WriteString(strconv.Itoa(l.EventSequence))
	if l.RecordHash != "" {
		b.WriteString("&record=")
		b.WriteString(l.RecordHash)
	}
	return b.String()
}

// Resolution is the result of resolving a locator against an in-memory bundle.
type Resolution struct {
	// Locator is the parsed locator that was resolved.
	Locator Locator
	// Index is the 0-based position of the record within the bundle.
	Index int
	// Seq is the referenced record's `seq` value (equal to Locator.EventSequence).
	Seq int
	// RecordHashMatched is true only when the locator supplied an optional
	// record hash and it matched the referenced record's stored hash. It is
	// false when no record hash was supplied.
	RecordHashMatched bool
}

// Resolve maps a parsed locator onto a loaded bundle. It performs no
// acquisition, no filesystem search, and no cryptographic verification: the
// caller is responsible for supplying the bundle and for any integrity check.
//
// The bundle head is the terminal record's stored hash. A mismatch, a missing
// sequence, or a record-hash mismatch returns the corresponding sentinel.
//
// Resolve returns the first record whose stored `seq` equals the locator's
// sequence. A bundle that (through tampering or corruption) contains duplicate
// sequence values is not rejected here: sequence uniqueness is an integrity
// property checked by bundle verification, not by the locator.
func Resolve(l Locator, b *bundle.Bundle) (Resolution, error) {
	if err := l.Validate(); err != nil {
		return Resolution{}, err
	}
	if b == nil || len(b.Records) == 0 {
		return Resolution{}, fmt.Errorf("%w: no bundle records to resolve against", ErrBundleUnavailable)
	}
	head := b.Records[len(b.Records)-1].Hash
	if l.BundleHeadHash != head {
		return Resolution{}, fmt.Errorf("%w: locator head %s does not match bundle head %s", ErrBundleUnavailable, l.BundleHeadHash, head)
	}
	for i := range b.Records {
		record := b.Records[i]
		if record.Event.Sequence != l.EventSequence {
			continue
		}
		res := Resolution{Locator: l, Index: i, Seq: l.EventSequence}
		if l.RecordHash != "" {
			if l.RecordHash != record.Hash {
				return Resolution{}, fmt.Errorf("%w: seq %d record hash %s does not match locator record %s", ErrRecordHashMismatch, l.EventSequence, record.Hash, l.RecordHash)
			}
			res.RecordHashMatched = true
		}
		return res, nil
	}
	return Resolution{}, fmt.Errorf("%w: seq %d is not present in the bundle", ErrEventNotFound, l.EventSequence)
}
