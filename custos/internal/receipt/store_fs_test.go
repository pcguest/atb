package receipt

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestReceiptStoreSaveAndGet(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory keeps receipt persistence tests isolated.
	baseDir := t.TempDir()
	// Added: The filesystem store writes one JSON document per receipt ID.
	store := NewFileSystemReceiptStore(baseDir)
	// Added: The sample receipt exercises every persisted field in the current schema.
	want := Receipt{
		ExportVersion: "atb.custody.bundle_export.v1",
		ReceiptID:     "sha256-abc123",
		BundleHash:    "abc123",
		SubmittedAt:   "2026-05-28T00:00:00Z",
		ProfileID:     "atb.profile.rag_answer",
		SubmitterRef:  "local-submit",
		VerifyReport:  json.RawMessage(`{"report_version":"verify.report.v1"}`),
	}

	if err := store.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Get(context.Background(), want.ReceiptID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("receipt mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestReceiptStoreNotFound(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory represents an empty receipt store.
	baseDir := t.TempDir()
	// Added: Missing receipts must return the typed sentinel for HTTP 404 mapping.
	store := NewFileSystemReceiptStore(baseDir)

	if _, err := store.Get(context.Background(), "missing"); !errors.Is(err, ErrReceiptNotFound) {
		t.Fatalf("expected ErrReceiptNotFound, got %v", err)
	}
}

func TestReceiptStoreList(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory keeps list ordering assertions deterministic.
	baseDir := t.TempDir()
	// Added: The filesystem store lists every valid receipt JSON file.
	store := NewFileSystemReceiptStore(baseDir)
	// Added: Distinct SubmittedAt values lock the expected ascending order.
	receipts := []Receipt{
		{
			ReceiptID:   "sha256-third",
			BundleHash:  "third",
			SubmittedAt: "2026-05-28T03:00:00Z",
		},
		{
			ReceiptID:   "sha256-first",
			BundleHash:  "first",
			SubmittedAt: "2026-05-28T01:00:00Z",
		},
		{
			ReceiptID:   "sha256-second",
			BundleHash:  "second",
			SubmittedAt: "2026-05-28T02:00:00Z",
		},
	}
	for _, receipt := range receipts {
		if err := store.Save(context.Background(), receipt); err != nil {
			t.Fatalf("Save(%s): %v", receipt.ReceiptID, err)
		}
	}

	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected three receipts, got %d", len(got))
	}
	if got[0].ReceiptID != "sha256-first" || got[1].ReceiptID != "sha256-second" || got[2].ReceiptID != "sha256-third" {
		t.Fatalf("receipts not sorted by SubmittedAt ascending: %+v", got)
	}
}
