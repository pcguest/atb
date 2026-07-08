// SPDX-License-Identifier: MIT
// Package identityevidence extracts caller-provided reviewer identity evidence
// from ATB events without making an identity-verification claim.
package identityevidence

import (
	"fmt"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
)

const Field = "identity_evidence"

// Evidence is the normalized reviewer identity evidence exposed in reports.
type Evidence struct {
	EventType         string `json:"event_type"`
	Sequence          int    `json:"sequence"`
	IdentityProvider  string `json:"identity_provider"`
	Subject           string `json:"subject"`
	AuthContext       string `json:"auth_context,omitempty"`
	AssertionType     string `json:"assertion_type"`
	AssertionDigest   string `json:"assertion_digest"`
	RawEvidenceDigest string `json:"raw_evidence_digest,omitempty"`
	Verification      string `json:"verification"`
}

// Extract returns well-formed identity evidence objects in bundle order.
func Extract(b *bundle.Bundle) []Evidence {
	if b == nil {
		return nil
	}
	var out []Evidence
	for _, record := range b.Records {
		data, ok := record.Event.Data.(map[string]any)
		if !ok {
			continue
		}
		raw, ok := data[Field].(map[string]any)
		if !ok {
			continue
		}
		evidence := Evidence{
			EventType:         record.Event.Type,
			Sequence:          record.Event.Sequence,
			IdentityProvider:  stringValue(raw, "identity_provider"),
			Subject:           stringValue(raw, "subject"),
			AuthContext:       stringValue(raw, "auth_context"),
			AssertionType:     stringValue(raw, "assertion_type"),
			AssertionDigest:   stringValue(raw, "assertion_digest"),
			RawEvidenceDigest: stringValue(raw, "raw_evidence_digest"),
			Verification:      "caller_provided_unverified",
		}
		if evidence.IdentityProvider == "" || evidence.Subject == "" ||
			evidence.AssertionType == "" || evidence.AssertionDigest == "" {
			continue
		}
		out = append(out, evidence)
	}
	return out
}

// Summary renders a concise, explicitly bounded reviewer identity statement.
func Summary(e Evidence) string {
	return fmt.Sprintf(
		"caller-provided identity evidence: provider=%s subject=%s assertion=%s digest=%s",
		e.IdentityProvider,
		e.Subject,
		e.AssertionType,
		e.AssertionDigest,
	)
}

func stringValue(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return strings.TrimSpace(value)
}
