// SPDX-License-Identifier: MIT
// Package jcs exposes ATB's RFC 8785 JSON Canonicalization Scheme (JCS)
// implementation for downstream consumers — the same canonicalisation the
// event chain hashes are built on. Downstream custody layers (e.g. a
// transparency log committing to receipt JSON) must use this rather than
// reimplementing RFC 8785, so leaf preimages stay reproducible against ATB's
// golden-tested canonical form.
package jcs

import (
	"github.com/pcguest/atb/v2/internal/canonicalize"
	"github.com/pcguest/atb/v2/internal/canonicalize/strict"
)

// Marshal returns the RFC 8785 canonical JSON encoding of v.
func Marshal(v interface{}) ([]byte, error) {
	return canonicalize.Marshal(v)
}

// MarshalRaw accepts an already-serialised JSON byte slice and returns its
// RFC 8785 canonical form.
func MarshalRaw(raw []byte) ([]byte, error) {
	return canonicalize.MarshalRaw(raw)
}

// MarshalRFC8785 returns the strict atb-jcs-rfc8785-1 canonical encoding of v.
// It is a new, non-production entry point; it is not used by hashing or bundle
// verification.
func MarshalRFC8785(v any) ([]byte, error) {
	return strict.CanonicalizeValue(v)
}

// MarshalRawRFC8785 returns the strict atb-jcs-rfc8785-1 canonical form of an
// already-serialised JSON byte slice. It is a new, non-production entry point;
// it is not used by hashing or bundle verification.
func MarshalRawRFC8785(raw []byte) ([]byte, error) {
	return strict.Canonicalize(raw)
}

// StrictErrorClass returns the stable strict-profile error class of err, or ""
// when err is nil or not produced by the strict canonicaliser.
func StrictErrorClass(err error) string {
	return strict.ErrorClass(err)
}
