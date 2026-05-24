// SPDX-License-Identifier: MIT
package corroboration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileReceiptAdapter_Fetch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "receipt.json")
	if err := os.WriteFile(path, []byte(`{"job_id":"job-1","status":"done"}`), 0o644); err != nil {
		t.Fatalf("write receipt: %v", err)
	}

	adapter := &FileReceiptAdapter{Path: path}
	rec, err := adapter.Fetch(context.Background(), "ref-1")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if rec.Adapter != "file-receipt" || rec.Digest == "" {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestFileReceiptAdapter_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "receipt.txt")
	if err := os.WriteFile(path, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write receipt: %v", err)
	}

	adapter := &FileReceiptAdapter{Path: path}
	if _, err := adapter.Fetch(context.Background(), "ref-1"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
