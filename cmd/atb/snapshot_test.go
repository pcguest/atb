package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
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
				Format:     verifyFormatText,
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
				Format:     verifyFormatText,
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
				Format:     verifyFormatJSON,
				DryRun:     true,
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
			if got != tc.want {
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
	if _, err := time.Parse(time.RFC3339, snapshotAt); err != nil {
		t.Fatalf("snapshot_at is not RFC3339: %v", err)
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

func TestSnapshotCreatesBundleWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "bundle.atb")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runSnapshot([]string{"first", "--bundle", path}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("runSnapshot() exit code = %d, want %d; stderr=%s", exitCode, exitSuccess, stderr.String())
	}

	loaded, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if len(loaded.Records) != 2 {
		t.Fatalf("unexpected bundle length: got %d want 2", len(loaded.Records))
	}
	if last := loaded.Records[len(loaded.Records)-1]; last.Event.Type != event.TypeSnapshot {
		t.Fatalf("unexpected event type: got %q want %q", last.Event.Type, event.TypeSnapshot)
	}
}

func writeSnapshotFixtureBundle(t *testing.T, path string) *bundle.Bundle {
	t.Helper()

	b := bundle.New()
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

	serialized, err := serializeBundleSnapshot(b)
	if err != nil {
		t.Fatalf("serialize bundle: %v", err)
	}
	sum := sha256.Sum256(serialized)
	return hex.EncodeToString(sum[:])
}
