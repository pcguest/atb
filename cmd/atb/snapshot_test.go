// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestSnapshotParseArgs(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "custom.atb")

	tests := []struct {
		name    string
		args    []string
		want    snapshotConfig
		wantErr bool
	}{
		{
			name: "defaults",
			args: []string{"release-cut"},
			want: snapshotConfig{
				Name:       "release-cut",
				BundlePath: bundle.DefaultPath(),
				Quiet:      false,
				Format:     formatText,
				DryRun:     false,
			},
		},
		{
			name: "bundle and quiet",
			args: []string{"release-cut", "--bundle", custom, "--quiet"},
			want: snapshotConfig{
				Name:       "release-cut",
				BundlePath: custom,
				Quiet:      true,
				Format:     formatText,
				DryRun:     false,
			},
		},
		{
			name: "json and dry-run",
			args: []string{"release-cut", "--format", "json", "--dry-run"},
			want: snapshotConfig{
				Name:       "release-cut",
				BundlePath: bundle.DefaultPath(),
				Quiet:      false,
				Format:     formatJSON,
				DryRun:     true,
			},
		},
		{
			name: "oversight attribution fields",
			args: []string{"release-cut", "--actor-id", "reviewer@example.test", "--actor-role=auditor", "--oversight-note", "Reviewed exception rationale."},
			want: snapshotConfig{
				Name:          "release-cut",
				BundlePath:    bundle.DefaultPath(),
				Format:        formatText,
				ActorID:       "reviewer@example.test",
				ActorRole:     "auditor",
				OversightNote: "Reviewed exception rationale.",
			},
		},
		{
			name:    "missing name",
			args:    nil,
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"release-cut", "--gate", "pass"},
			wantErr: true,
		},
		{
			name:    "too many names",
			args:    []string{"one", "two"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSnapshotArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected config: got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestSnapshotAppendsRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.atb", "bundle.atb")
	initial := writeSnapshotFixtureBundle(t, path)
	expectedHash := hashSerializedBundle(t, initial)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{"my-snap", "--bundle", path}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitSuccess, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if want := "snapshot my-snap  records=2  hash=" + expectedHash[:12] + "..."; !strings.Contains(stdout.String(), want) {
		t.Fatalf("runSnapshot() stdout missing %q:\n%s", want, stdout.String())
	}

	loaded, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if len(loaded.Records) != 3 {
		t.Fatalf("unexpected bundle length: got %d want 3", len(loaded.Records))
	}
	last := loaded.Records[len(loaded.Records)-1]
	if last.Event.Type != event.TypeSnapshot {
		t.Fatalf("unexpected event type: got %q want %q", last.Event.Type, event.TypeSnapshot)
	}

	data, ok := last.Event.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected snapshot event data map, got %T", last.Event.Data)
	}
	if got := data["name"]; got != "my-snap" {
		t.Fatalf("unexpected snapshot name: got %v want %q", got, "my-snap")
	}
	if got := data["bundle_hash"]; got != expectedHash {
		t.Fatalf("unexpected bundle hash: got %v want %q", got, expectedHash)
	}
	if got := data["record_count"]; got != float64(2) {
		t.Fatalf("unexpected record count: got %v want %d", got, 2)
	}
	snapshotAt, ok := data["snapshot_at"].(string)
	if !ok {
		t.Fatalf("expected snapshot_at string, got %T", data["snapshot_at"])
	}
	if _, err := time.Parse(time.RFC3339Nano, snapshotAt); err != nil {
		t.Fatalf("snapshot_at is not RFC3339Nano: %v", err)
	}
}

func TestSnapshotOversightFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.atb", "bundle.atb")
	writeSnapshotFixtureBundle(t, path)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{
		"oversight",
		"--bundle", path,
		"--actor-id", "reviewer@example.test",
		"--actor-role", "auditor",
		"--oversight-note", "Reviewed Article 14 handoff.",
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitSuccess, stderr.String())
	}

	data := loadLastSnapshotData(t, path)
	if got := data["actor_id"]; got != "reviewer@example.test" {
		t.Fatalf("actor_id = %v, want reviewer@example.test", got)
	}
	if got := data["actor_role"]; got != "auditor" {
		t.Fatalf("actor_role = %v, want auditor", got)
	}
	if got := data["oversight_note"]; got != "Reviewed Article 14 handoff." {
		t.Fatalf("oversight_note = %v, want note", got)
	}
}

func TestSnapshotOversightFieldsOmittedWhenEmpty(t *testing.T) {
	raw, err := json.Marshal(snapshotEventData{
		Name:        "plain",
		BundleHash:  strings.Repeat("a", 64),
		RecordCount: 2,
		SnapshotAt:  "2026-04-09T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal snapshot event data: %v", err)
	}
	for _, field := range []string{"actor_id", "actor_role", "oversight_note"} {
		if bytes.Contains(raw, []byte(field)) {
			t.Fatalf("unexpected %s in marshalled data: %s", field, raw)
		}
	}
}

func TestSnapshotOversightNoteValidation(t *testing.T) {
	tests := []struct {
		name    string
		note    string
		wantErr bool
	}{
		{name: "too long", note: strings.Repeat("a", 513), wantErr: true},
		{name: "null byte", note: "reviewed\x00rejected", wantErr: true},
		{name: "empty optional", note: ""},
		{name: "tab and newline allowed", note: "line one\n\tline two"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOversightNote(tc.note)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSnapshotOversightNoteInvalidExitsUserError(t *testing.T) {
	tests := []struct {
		name string
		note string
	}{
		{name: "too long", note: strings.Repeat("a", 513)},
		{name: "null byte", note: "reviewed\x00rejected"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "run.atb", "bundle.atb")
			writeSnapshotFixtureBundle(t, path)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runSnapshot([]string{"invalid-note", "--bundle", path, "--oversight-note", tc.note}, &stdout, &stderr)
			if exitCode != exitUserError {
				t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitUserError, stderr.String())
			}
			if !strings.Contains(stderr.String(), "oversight note") {
				t.Fatalf("stderr missing oversight note detail: %q", stderr.String())
			}
		})
	}
}

func TestSnapshotActorFieldsWithEmptyNote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.atb", "bundle.atb")
	writeSnapshotFixtureBundle(t, path)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{
		"actor-only",
		"--bundle", path,
		"--actor-id", "human-123",
		"--actor-role", "operator",
		"--oversight-note", "",
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitSuccess, stderr.String())
	}

	data := loadLastSnapshotData(t, path)
	if got := data["actor_id"]; got != "human-123" {
		t.Fatalf("actor_id = %v, want human-123", got)
	}
	if got := data["actor_role"]; got != "operator" {
		t.Fatalf("actor_role = %v, want operator", got)
	}
	if _, ok := data["oversight_note"]; ok {
		t.Fatalf("oversight_note should be omitted when empty: %+v", data)
	}
}

func TestSnapshotQuiet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.atb", "bundle.atb")
	writeSnapshotFixtureBundle(t, path)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{"my-snap", "--bundle", path, "--quiet"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitSuccess, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
}

func TestSnapshotJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.atb")
	writeSnapshotFixtureBundle(t, path)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{"my-snap", "--bundle", path, "--format", "json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitSuccess, stderr.String())
	}

	var got mutationResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode mutation json: %v", err)
	}
	if got.EventType != event.TypeSnapshot {
		t.Fatalf("unexpected event type: got %q want %q", got.EventType, event.TypeSnapshot)
	}
	if got.Action != "snapshot" {
		t.Fatalf("unexpected action: got %q want %q", got.Action, "snapshot")
	}
}

func TestSnapshotMissingBundleExitsUserError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "bundle.atb")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{"first", "--bundle", path}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitUserError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "create one first") {
		t.Fatalf("expected 'create one first' guidance, got stderr=%q", stderr.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected no bundle at %s, got err=%v", path, err)
	}
}

func TestSnapshotEventUsesNanoPrecisionTimestamp(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	writeSnapshotFixtureBundle(t, bundlePath)

	var stdout, stderr bytes.Buffer
	exit := runSnapshot([]string{"nano", "--bundle", bundlePath}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	last := b.Records[len(b.Records)-1]
	if last.Event.Type != event.TypeSnapshot {
		t.Fatalf("last event type = %q, want %q", last.Event.Type, event.TypeSnapshot)
	}
	parsed, err := time.Parse(time.RFC3339Nano, last.Event.Timestamp)
	if err != nil {
		t.Fatalf("parse RFC3339Nano timestamp %q: %v", last.Event.Timestamp, err)
	}
	// Sanity-check the format expanded the precision: a base RFC3339 string
	// has length 20 ("2006-01-02T15:04:05Z"); RFC3339Nano with non-zero
	// nanoseconds is longer. We accept either, but if the local clock has
	// nanosecond precision we expect a fractional component.
	if parsed.Nanosecond() != 0 && !strings.Contains(last.Event.Timestamp, ".") {
		t.Fatalf("timestamp %q lacks fractional component despite Nanosecond()=%d", last.Event.Timestamp, parsed.Nanosecond())
	}
}

func TestSnapshotsOneMillisecondApartHaveDistinctSnapshotAt(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.atb")
	writeSnapshotFixtureBundle(t, bundlePath)

	t0 := time.Date(2026, 4, 27, 12, 0, 0, 1_000_000, time.UTC)
	clockTimes := []time.Time{t0, t0.Add(1 * time.Millisecond)}
	idx := 0
	clock := func() time.Time {
		ts := clockTimes[idx]
		idx++
		return ts
	}

	cfg := snapshotConfig{
		Name:            "first",
		BundlePath:      bundlePath,
		Format:          formatText,
		RequireExisting: true,
		snapshotClock:   clock,
	}
	first, err := appendSnapshot(cfg)
	if err != nil {
		t.Fatalf("first appendSnapshot: %v", err)
	}

	cfg.Name = "second"
	second, err := appendSnapshot(cfg)
	if err != nil {
		t.Fatalf("second appendSnapshot: %v", err)
	}

	if first.Data.SnapshotAt == second.Data.SnapshotAt {
		t.Fatalf("expected distinct SnapshotAt values, both = %q", first.Data.SnapshotAt)
	}
	if _, err := time.Parse(time.RFC3339Nano, first.Data.SnapshotAt); err != nil {
		t.Fatalf("first SnapshotAt not RFC3339Nano: %v", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, second.Data.SnapshotAt); err != nil {
		t.Fatalf("second SnapshotAt not RFC3339Nano: %v", err)
	}
}

func TestCaptureRunCreateNoticeAppearsOnce(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "fresh.atb")

	var stdout, stderr bytes.Buffer
	exit := runCaptureRun([]string{
		"--bundle", bundlePath,
		"--snapshot", "after_run",
		"--",
		"sh", "-c", "true",
	}, bytes.NewBuffer(nil), &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("runCaptureRun exit = %d, want %d (stderr=%q)", exit, exitSuccess, stderr.String())
	}
	count := strings.Count(stderr.String(), "created new bundle")
	if count != 1 {
		t.Fatalf("expected 'created new bundle' exactly once, got %d (stderr=%q)", count, stderr.String())
	}
}

func writeSnapshotFixtureBundle(t *testing.T, path string) *bundle.Bundle {
	t.Helper()

	b := newTestBundle(t)
	if err := b.AppendWithOptions(event.TypeDevSession, "snapshot-fixture", &bundle.AppendOptions{
		Timestamp: "2026-03-28T03:04:05Z",
	}); err != nil {
		t.Fatalf("append dev session: %v", err)
	}
	if err := b.Save(path); err != nil {
		t.Fatalf("save bundle: %v", err)
	}
	return b
}

func hashSerializedBundle(t *testing.T, b *bundle.Bundle) string {
	t.Helper()

	hash, err := verifypkg.SnapshotBundleHash(b.Records)
	if err != nil {
		t.Fatalf("snapshot bundle hash: %v", err)
	}
	return hash
}

func loadLastSnapshotData(t *testing.T, path string) map[string]any {
	t.Helper()

	loaded, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	last := loaded.Records[len(loaded.Records)-1]
	if last.Event.Type != event.TypeSnapshot {
		t.Fatalf("last event type = %q, want %q", last.Event.Type, event.TypeSnapshot)
	}
	data, ok := last.Event.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected snapshot event data map, got %T", last.Event.Data)
	}
	return data
}
