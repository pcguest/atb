// SPDX-License-Identifier: MIT
package bundle

import "errors"

// ErrTamper is returned when the bundle's hash chain or sequence numbering
// fails to verify. A failure means the presented records could not be verified
// against their recorded hash chain; it does not by itself establish cause or
// intent. Callers should treat any error matchable via errors.Is(err,
// ErrTamper) as a hard integrity failure.
var ErrTamper = errors.New("bundle: integrity verification failed")

// ErrMalformed is returned when a bundle is structurally invalid in a way
// the reader cannot recover from — including a manifest version that this
// build of atb does not understand. Wrapped errors should retain
// errors.Is(err, ErrMalformed) for callers that want to distinguish
// malformed bundles from integrity failures.
var ErrMalformed = errors.New("bundle: malformed")

// ErrResourceLimit is returned when an untrusted bundle exceeds a reader
// byte, record-count, or per-record limit. It is separate from ErrMalformed so
// callers can report that the input may be valid but is unsafe to load under
// the configured policy.
var ErrResourceLimit = errors.New("bundle: resource limit exceeded")

// ErrNoManifest is returned when an operation requires a manifest record
// but the bundle has no records at all.
var ErrNoManifest = errors.New("bundle: no manifest record")

// ErrNotABundle is returned when a file parses as NDJSON but its first
// record is not the reserved manifest event type.
var ErrNotABundle = errors.New("bundle: not a bundle file")

// ErrBundleLocked is declared in lock.go (kept there to avoid touching the
// lock implementation in this change). It is part of the same sentinel
// vocabulary as the errors above.
