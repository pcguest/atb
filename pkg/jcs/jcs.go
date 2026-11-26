// SPDX-License-Identifier: MIT
// Package jcs exposes the legacy and strict JSON canonicalisation profiles for
// downstream consumers.
//
// Marshal and MarshalRaw are the legacy event-chain canonicalisation: this is
// the canonical form that ATB hashes, bundle signing, and verification are
// built on. Downstream custody layers (e.g. a transparency log committing to
// receipt JSON) must use these rather than reimplementing RFC 8785, so leaf
// preimages stay reproducible against ATB's golden-tested canonical form.
//
// MarshalRFC8785 and MarshalRawRFC8785 implement the frozen
// atb-jcs-rfc8785-1 profile: RFC 8785 plus the ATB safe-integer
// interoperability restriction. MarshalRFC8785V2 and MarshalRawRFC8785V2 add
// the atb-jcs-rfc8785-2 canonical-output-closed policy. All four are
// non-production entry points and are not used by event-chain hashing or
// bundle verification. Their output must not be substituted for the legacy
// canonicalisation of an existing record or for a v1 historical hash.
package jcs

import (
	"github.com/pcguest/atb/v2/internal/canonicalize"
	"github.com/pcguest/atb/v2/internal/canonicalize/strict"
)

// Marshal returns the legacy event-chain canonical JSON encoding of v. This is
// the profile ATB hashing and bundle verification use.
func Marshal(v interface{}) ([]byte, error) {
	return canonicalize.Marshal(v)
}

// MarshalRaw accepts an already-serialised JSON byte slice and returns its
// legacy event-chain canonical form. This is the profile ATB hashing and bundle
// verification use.
func MarshalRaw(raw []byte) ([]byte, error) {
	return canonicalize.MarshalRaw(raw)
}

// MarshalRFC8785 returns the frozen atb-jcs-rfc8785-1 canonical encoding of v:
// RFC 8785 plus the ATB safe-integer restriction. It is a non-production entry
// point; it is not used by hashing or bundle verification and must not be
// substituted for a v1 historical hash.
func MarshalRFC8785(v any) ([]byte, error) {
	return strict.CanonicalizeValue(v)
}

// MarshalRawRFC8785 returns the frozen atb-jcs-rfc8785-1 canonical form of an
// already-serialised JSON byte slice: RFC 8785 plus the ATB safe-integer
// restriction. It is a non-production entry point; it is not used by hashing
// or bundle verification and must not be substituted for a v1 historical hash.
func MarshalRawRFC8785(raw []byte) ([]byte, error) {
	return strict.Canonicalize(raw)
}

// MarshalRFC8785V2 returns the atb-jcs-rfc8785-2 canonical encoding of v. It
// is an additive, non-production profile: it retains the v1 plain-integer
// input cap and also rejects canonical fixed-notation outputs outside
// ±(2^53 - 1). It is not used by hashing or bundle verification and must not
// be substituted for v1 historical hashes.
func MarshalRFC8785V2(v any) ([]byte, error) {
	return strict.CanonicalizeValueV2(v)
}

// MarshalRawRFC8785V2 returns the atb-jcs-rfc8785-2 canonical form of an
// already-serialised JSON byte slice. It is an additive, non-production
// profile: it retains the v1 plain-integer input cap and also rejects
// canonical fixed-notation outputs outside ±(2^53 - 1). It is not used by
// hashing or bundle verification and must not be substituted for v1
// historical hashes.
func MarshalRawRFC8785V2(raw []byte) ([]byte, error) {
	return strict.CanonicalizeV2(raw)
}

// StrictErrorClass returns the stable strict-profile error class of err, or ""
// when err is nil or not produced by the strict canonicaliser. It reports v1
// and v2 classes; the class vocabulary is shared.
func StrictErrorClass(err error) string {
	return strict.ErrorClass(err)
}
