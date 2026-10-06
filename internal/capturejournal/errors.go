// SPDX-License-Identifier: MIT
package capturejournal

import "errors"

// ErrJournalCorrupted indicates the journal failed chain or JSON validation.
var ErrJournalCorrupted = errors.New("capturejournal: journal corrupted")

// ErrJournalVersionUnsupported indicates an entry format_version other than
// JournalFormatVersion was found on append or read.
var ErrJournalVersionUnsupported = errors.New("capturejournal: unsupported journal format version")

// ErrJournalEntryTooLarge indicates an entry exceeded the per-line limit.
var ErrJournalEntryTooLarge = errors.New("capturejournal: journal entry exceeds maximum size")

// ErrJournalClosed indicates an append was attempted on a closed journal.
var ErrJournalClosed = errors.New("capturejournal: journal is closed")

// ErrJournalFailed indicates an append was attempted after a prior write or
// fsync failure left the durable file inconsistent with the in-memory tail.
var ErrJournalFailed = errors.New("capturejournal: journal failed after a write error")
