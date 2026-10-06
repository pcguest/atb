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
	// A legacy checkpoint with no recorded version/incarnation remains usable.
	if err := got.Validate("proxy", "stream", "atb.intercept", "", ""); err != nil {
		t.Fatalf("empty expected version/incarnation should not fail: %v", err)
	}
}

func TestPlanReconciliationSuppressesIntraPassDuplicate(t *testing.T) {
	acq := &event.AcquisitionInfo{
		Mode:           "live",
		SourceSystem:   "proxy",
		SourceRecordID: "session:1",
		SourceDigest:   "digest-a",
		AcquiredAt:     "2026-01-01T00:00:00Z",
	}
	events := []importEvent{
		{typ: "atb.llm.request", acquisition: acq},
		{typ: "atb.llm.request", acquisition: acq},
	}

	plan, err := planReconciliation(events, NewReconciler(nil), true)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(plan.actions))
	}
	if plan.actions[0].skip {
		t.Fatalf("first occurrence should be appended")
	}
	if !plan.actions[1].skip {
		t.Fatalf("intra-pass duplicate must be suppressed, not appended twice")
	}
}

func TestPlanReconciliationSurfacesIntraPassChange(t *testing.T) {
	first := &event.AcquisitionInfo{
		Mode:           "live",
		SourceSystem:   "proxy",
		SourceRecordID: "session:1",
		SourceDigest:   "digest-a",
		AcquiredAt:     "2026-01-01T00:00:00Z",
	}
	second := &event.AcquisitionInfo{
		Mode:           "live",
		SourceSystem:   "proxy",
		SourceRecordID: "session:1",
		SourceDigest:   "digest-b",
		AcquiredAt:     "2026-01-01T00:00:01Z",
	}
	events := []importEvent{
		{typ: "atb.llm.request", acquisition: first},
		{typ: "atb.llm.request", acquisition: second},
	}

	plan, err := planReconciliation(events, NewReconciler(nil), true)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.actions[1].skip {
		t.Fatalf("changed representation within a pass must be appended")
	}
	if plan.actions[1].finding == nil {
		t.Fatalf("changed representation within a pass must surface a finding")
	}
	if plan.actions[1].finding.PreviousDigest != "digest-a" || plan.actions[1].finding.CurrentDigest != "digest-b" {
		t.Fatalf("finding digests = %+v", plan.actions[1].finding)
	}
}
