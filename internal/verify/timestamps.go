// SPDX-License-Identifier: MIT
package verify

import (
	"fmt"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

// ValidateTimestamps reports timestamp format and causal-order violations for
// the provided records. Missing timestamps are ignored.
func ValidateTimestamps(records []bundle.Record) []string {
	violations := make([]string, 0)
	var previous time.Time
	previousSet := false

	for _, record := range records {
		if record.Event.Type == bundle.ManifestEventType {
			continue
		}

		raw := strings.TrimSpace(record.Event.Timestamp)
		if raw == "" {
			continue
		}

		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			violations = append(violations,
				fmt.Sprintf("timestamp validation: seq %d (%s) has invalid RFC 3339 timestamp %q",
					record.Event.Sequence, record.Event.Type, raw))
			continue
		}

		if previousSet && parsed.Before(previous) {
			violations = append(violations,
				fmt.Sprintf("timestamp validation: seq %d (%s) timestamp %q is earlier than the preceding timestamp %q",
					record.Event.Sequence, record.Event.Type, raw, previous.Format(time.RFC3339)))
		}

		previous = parsed
		previousSet = true
	}

	return violations
}
