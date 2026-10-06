// SPDX-License-Identifier: MIT
package acquisition

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/event"
)

func TestCheckpointAdapterVersionAndIncarnationMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cp.json")

	cp := &Checkpoint{
		SourceSystem:      "proxy",
		AcquisitionStream: "stream",
		Adapter:           "atb.intercept",
		AdapterVersion:    "1.0.0",
		SourceIncarnation: "inc-1",
	}
	if err := cp.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := got.Validate("proxy", "stream", "atb.intercept", "1.0.0", "inc-1"); err != nil {
		t.Fatalf("matching validate: %v", err)
	}
	if err := got.Validate("proxy", "stream", "atb.intercept", "2.0.0", "inc-1"); !errors.Is(err, ErrCheckpointAdapterMismatch) {
		t.Fatalf("adapter version mismatch: got %v", err)
	}
	if err := got.Validate("proxy", "stream", "atb.intercept", "1.0.0", "inc-2"); !errors.Is(err, ErrCheckpointIncarnationMismatch) {
		t.Fatalf("incarnation mismatch: got %v", err)
	}
	// A caller that supplies no version/incarnation must not fail a checkpoint
	// that has them (the other side of the && guard).
	if err := got.Validate("proxy", "stream", "atb.intercept", "", ""); err != nil {
		t.Fatalf("empty expected version/incarnation should not fail: %v", err)
	}
}

func TestCheckpointLegacyWithoutVersionOrIncarnationRemainsUsable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")

	// A legacy checkpoint recorded no adapter version or incarnation.
	legacy := &Checkpoint{
		SourceSystem:      "chatlog",
		AcquisitionStream: "stream",
		Adapter:           "atb.chatlog.generic-jsonl",
	}
	if err := legacy.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Presenting non-empty expected values must still accept the legacy record:
	// a missing recorded value is unknown, not a mismatch.
	if err := got.Validate("chatlog", "stream", "atb.chatlog.generic-jsonl", "2.0.0", "inc-2"); err != nil {
		t.Fatalf("legacy checkpoint should remain usable: %v", err)
	}
}

// A single source record can legitimately emit several events that share one
// identity and digest (chatlog request/model/tool/output/response). All of them
// must be appended together; none may be dropped as an intra-pass duplicate.
func TestPlanReconciliationKeepsAllEventsOfOneRecord(t *testing.T) {
	acq := &event.AcquisitionInfo{
		Mode:           "retrospective",
		SourceSystem:   "chatlog",
		SourceRecordID: "exchange:1",
		SourceDigest:   "digest-a",
		AcquiredAt:     "2026-01-01T00:00:00Z",
	}
	events := []importEvent{
		{typ: "atb.llm.request", acquisition: acq},
		{typ: "ai.model.invoked", acquisition: acq},
		{typ: "atb.tool.call", acquisition: acq},
		{typ: "atb.llm.response", acquisition: acq},
	}

	for _, reconcile := range []bool{false, true} {
		plan, err := planReconciliation(events, NewReconciler(nil), reconcile)
		if err != nil {
			t.Fatalf("plan (reconcile=%v): %v", reconcile, err)
		}
		if len(plan.actions) != len(events) {
			t.Fatalf("actions = %d, want %d", len(plan.actions), len(events))
		}
		for i, action := range plan.actions {
			if action.skip {
				t.Fatalf("event %d (reconcile=%v) was skipped; all events of one new record must be appended", i, reconcile)
			}
		}
	}
}

// Across passes, an unchanged record is skipped as a whole: all of its events
// share one decision, so re-importing produces no duplicates.
func TestPlanReconciliationSkipsUnchangedRecordAsAWhole(t *testing.T) {
	acq := &event.AcquisitionInfo{
		Mode:           "retrospective",
		SourceSystem:   "chatlog",
		SourceRecordID: "exchange:1",
		SourceDigest:   "digest-a",
		AcquiredAt:     "2026-01-01T00:00:00Z",
	}
	reconciler := NewReconciler(nil)
	// Seed the reconciler by reconciling once, then plan the same record again.
	if _, err := reconciler.Reconcile(event.SourceIdentity{System: "chatlog", RecordID: "exchange:1", Derived: true}, "digest-a", acq.AcquiredAt, acq); err != nil {
		t.Fatalf("seed reconcile: %v", err)
	}

	events := []importEvent{
		{typ: "atb.llm.request", acquisition: acq},
		{typ: "atb.llm.response", acquisition: acq},
	}
	plan, err := planReconciliation(events, reconciler, true)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	for i, action := range plan.actions {
		if !action.skip {
			t.Fatalf("event %d of an unchanged record must be skipped", i)
		}
	}
}
