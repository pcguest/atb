// SPDX-License-Identifier: MIT
// Package locator exposes ATB's semantic evidence locator grammar for external
// consumers: a versioned, canonical reference to a single evidence record
// inside one ATB bundle.
//
// This package exposes the grammar only. It does not acquire bundles and does
// not resolve locators; resolution remains an ATB-internal operation. A locator
// identifies evidence. It does not grant authority, establish factual truth, or
// imply governance approval.
package locator

import internallocator "github.com/pcguest/atb/v2/internal/locator"

// GrammarVersion is the locator grammar version supported by this build.
const GrammarVersion = internallocator.GrammarVersion

// ErrMalformed is returned when a locator is syntactically invalid.
var ErrMalformed = internallocator.ErrMalformed

// ErrUnsupportedVersion is returned when a locator declares a grammar version
// this build does not understand.
var ErrUnsupportedVersion = internallocator.ErrUnsupportedVersion

// ErrBundleUnavailable is returned when a locator cannot be resolved against
// the supplied bundle.
var ErrBundleUnavailable = internallocator.ErrBundleUnavailable

// ErrEventNotFound is returned when the locator's event sequence is not
// present in the resolved bundle.
var ErrEventNotFound = internallocator.ErrEventNotFound

// ErrRecordHashMismatch is returned when an optional record hash is present
// and does not match the referenced record's stored hash.
var ErrRecordHashMismatch = internallocator.ErrRecordHashMismatch

// Locator is a parsed semantic reference to one evidence record.
type Locator = internallocator.Locator

// Parse parses and validates a locator string, returning a Locator whose
// String method emits the canonical serialised form.
//
// Parsing is strict: surrounding whitespace, percent-encoding, fragments,
// unknown query parameters, duplicate parameters, and non-canonical hashes are
// rejected so that a locator has exactly one canonical representation.
func Parse(raw string) (Locator, error) {
	return internallocator.Parse(raw)
}

// Format constructs, validates, and serialises a locator in its canonical
// form from the bundle head hash, event sequence, and optional record hash.
func Format(bundleHeadHash string, eventSequence int64, recordHash string) (string, error) {
	loc := Locator{
		BundleHeadHash: bundleHeadHash,
		EventSequence:  eventSequence,
		RecordHash:     recordHash,
	}
	if err := loc.Validate(); err != nil {
		return "", err
	}
	return loc.String(), nil
}
