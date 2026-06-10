// SPDX-License-Identifier: MIT
// Package jcs exposes ATB's RFC 8785 JSON Canonicalization Scheme (JCS)
// implementation for downstream consumers — the same canonicalisation the
// event chain hashes are built on. Downstream custody layers (e.g. a
// transparency log committing to receipt JSON) must use this rather than
// reimplementing RFC 8785, so leaf preimages stay reproducible against ATB's
// golden-tested canonical form.
package jcs

import "github.com/pcguest/atb/internal/canonicalize"

// Marshal returns the RFC 8785 canonical JSON encoding of v.
func Marshal(v interface{}) ([]byte, error) {
	return canonicalize.Marshal(v)
}

// MarshalRaw accepts an already-serialised JSON byte slice and returns its
// RFC 8785 canonical form.
func MarshalRaw(raw []byte) ([]byte, error) {
	return canonicalize.MarshalRaw(raw)
}
