// SPDX-License-Identifier: MIT
package capturejournal

import "errors"

// ErrJournalCorrupted indicates the journal failed chain or JSON validation.
var ErrJournalCorrupted = errors.New("capturejournal: journal corrupted")

// ErrJournalVersionUnsupported indicates an unsupported entry format version.
var ErrJournalVersionUnsupported = errors.New("capturejournal: unsupported journal format version")
