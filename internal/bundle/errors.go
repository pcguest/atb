// SPDX-License-Identifier: MIT
package bundle

import "errors"

// ErrTamper is returned when the bundle's hash chain or sequence numbering
// fails to verify, indicating the bundle has been altered after it was
// written. Callers should treat any error matchable via errors.Is(err,
// ErrTamper) as a hard integrity failure.
var ErrTamper = errors.New("bundle: tampered")

// ErrMalformed is returned when a bundle is structurally invalid in a way
// the reader cannot recover from — including a manifest version that this
// build of atb does not understand. Wrapped errors should retain
// errors.Is(err, ErrMalformed) for callers that want to distinguish
// malformed bundles from tamper detection.
var ErrMalformed = errors.New("bundle: malformed")

// ErrNoManifest is returned when an operation requires a manifest record
// but the bundle has no records at all.
var ErrNoManifest = errors.New("bundle: no manifest record")

// ErrNotABundle is returned when a file parses as NDJSON but its first
// record is not the reserved manifest event type.
var ErrNotABundle = errors.New("bundle: not a bundle file")

// ErrBundleLocked is declared in lock.go (kept there to avoid touching the
// lock implementation in this change). It is part of the same sentinel
// vocabulary as the errors above.
