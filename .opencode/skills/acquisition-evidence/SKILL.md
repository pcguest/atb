---
name: acquisition-evidence
description: "Use when working on acquisition continuity — source provenance, source digests, checkpoints, reconciliation, retrospective vs live capture, and re-import. Trigger on: import chatlog/otel, --continue, --reconcile, --checkpoint, SourceIdentity, AcquisitionInfo, source_record_changed, SOURCE_RECORD_CHANGED, checkpoint mismatch/staleness, dedupe, source mutation."
---

# Acquisition Evidence

## Purpose

Preserve and relate how evidence was acquired, without claiming more than the
capture supports.

## Meaning (do not overclaim)

- `source_digest` establishes **source-representation identity/change**, NOT truth.
- A checkpoint is **operational state**, NOT evidence truth.
- `retrospective` means acquired after the original occurrence, NOT witnessed live.
- Reconciliation results: NEW / UNCHANGED / CHANGED / UNKNOWN.
- A CHANGED result yields a bounded `atb.acquisition.finding`
  (`source_record_changed`) with previous/current digests and acquisition times.

## When to invoke

- Editing `internal/acquisition/*`
- Editing `cmd/atb/import*.go`
- Changing acquisition fields on `internal/event/event.go`
- Adding viewer/API exposure of acquisition provenance

## Bounded procedure

1. Keep SourceIdentity stable and explicit (`system` + `record_id`, `derived` flag).
2. Ensure digests are computed from the raw source representation before translation.
3. With `--reconcile`: UNCHANGED records are skipped (idempotent); CHANGED
   records append a finding and the new representation; NEW/UNKNOWN records
   append, with UNKNOWN counted separately. A plain import (no `--reconcile`)
   appends every record.
4. Checkpoint save is atomic; on `--continue`/`--reconcile`, load validates
   source system/stream/adapter.
5. Never fabricate raw-source availability. Report availability honestly.
6. Legacy/bundle compatibility: absent acquisition fields must degrade gracefully.

## Stop conditions

- Stop if a change would make verification depend on checkpoint state.
- Stop if a "changed" record would silently overwrite prior evidence.
- Stop if raw source is implied available when it was not captured.

## Canonical references

- docs/integrations/chatlog-import.md
- docs/capture/overview.md
- docs/concepts/evidence-model.md
- internal/event/event.go (AcquisitionInfo, CheckpointInfo, SourceIdentity)

## Tests / checks

- Three-pass fixture: dedupe, changed source, finding idempotence
- Checkpoint restart + mismatch
- Legacy bundle, mixed retrospective/live, tamper detection
- `go test ./internal/acquisition/...`, `make test-golden`

## Prohibited interpretations

- A matching digest does not mean the source is authentic or true.
- A checkpoint does not prove capture completeness.
- Historical import is not live witness.
