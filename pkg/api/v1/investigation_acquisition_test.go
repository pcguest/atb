// SPDX-License-Identifier: MIT
package apiv1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// TestInvestigationAcquisitionProvenance verifies that acquisition provenance is
// exposed through the existing investigation surfaces (Evidence records,
// Findings, Timeline) without a new top-level page, and that it stays bounded:
// raw source is reported unavailable and checkpoint is labelled operational.
func TestInvestigationAcquisitionProvenance(t *testing.T) {
	bundlePath, b := createRichTestBundle(t)

	acq := &event.AcquisitionInfo{
		Mode:            "retrospective",
		SourceSystem:    "chatlog",
		SourceRecordID:  "r2",
		SourceTimestamp: "2026-01-01T10:00:02Z",
		AcquiredAt:      "2026-01-02T09:00:00Z",
		SourceDigest:    "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Adapter:         "atb.chatlog.generic-jsonl",
		AdapterVersion:  "1.0.0",
		Checkpoint: &event.CheckpointInfo{
			SourceSystem:      "chatlog",
			AcquisitionStream: "pass2.jsonl",
			Position:          "r2",
			ObservedAt:        "2026-01-02T09:00:00Z",
			Adapter:           "atb.chatlog.generic-jsonl",
			AdapterVersion:    "1.0.0",
		},
	}
	if err := b.AppendWithOptions(event.TypeAIRequestReceived, map[string]any{
		"request_id": "r2", "input_digest": "sha256:in",
	}, &bundle.AppendOptions{Timestamp: "2026-01-01T10:00:02Z", Acquisition: acq}); err != nil {
		t.Fatalf("append acquired record: %v", err)
	}

	if err := b.AppendWithOptions(event.TypeAcquisitionFinding, map[string]any{
		"finding_type":         "source_record_changed",
		"source_system":        "chatlog",
		"source_record_id":     "r2",
		"previous_digest":      "sha256:prev",
		"current_digest":       "sha256:curr",
		"previous_acquired_at": "2026-01-01T09:00:00Z",
		"current_acquired_at":  "2026-01-02T09:00:00Z",
		"adapter":              "atb.chatlog.generic-jsonl",
	}, &bundle.AppendOptions{Timestamp: "2026-01-02T09:00:00Z"}); err != nil {
		t.Fatalf("append finding: %v", err)
	}

	_, handler := buildTestAPIServer(t, APIConfig{BundlePath: bundlePath, Bundle: b})

	get := func(path string) string {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status %d: %s", path, rr.Code, rr.Body.String())
		}
		return rr.Body.String()
	}

	// Evidence records carry bounded acquisition provenance.
	events := get("/api/v1/bundle/events?limit=200")
	for _, want := range []string{
		`"acquisition":{`,
		`"mode":"retrospective"`,
		`"source_system":"chatlog"`,
		`"source_record_id":"r2"`,
		`"source_digest":"sha256:aaaa`,
		`"adapter":"atb.chatlog.generic-jsonl"`,
		`"raw_source_available":false`,
		`"checkpoint_position":"r2"`,
		`"checkpoint_status":"recorded"`,
	} {
		if !strings.Contains(events, want) {
			t.Fatalf("bundle/events missing %q", want)
		}
	}

	// Findings surface includes the bounded acquisition finding.
	findings := get("/api/v1/investigation/findings")
	for _, want := range []string{
		`"acquisition_findings":[`,
		`"flag":"source_record_changed"`,
		`"source_record_id":"r2"`,
		`"previous_digest":"sha256:prev"`,
		`"current_digest":"sha256:curr"`,
		`"boundedness":"bounded"`,
		`"what_atb_cannot_conclude":"ATB does not establish why the representation changed`,
	} {
		if !strings.Contains(findings, want) {
			t.Fatalf("investigation/findings missing %q: %s", want, findings)
		}
	}

	// Timeline labels the acquisition finding without implying causality.
	timeline := get("/api/v1/investigation/timeline")
	if !strings.Contains(timeline, `"label":"Source record changed"`) {
		t.Fatalf("timeline missing acquisition label: %s", timeline)
	}
	if !strings.Contains(timeline, `"family":"acquisition"`) {
		t.Fatalf("timeline missing acquisition family: %s", timeline)
	}
}
