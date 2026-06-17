package receipt

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestReceiptStoreSaveAndGet(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory keeps receipt persistence tests isolated.
	baseDir := t.TempDir()
	// Added: The filesystem store writes one JSON document per receipt ID.
	store := NewFileSystemReceiptStore(baseDir, RetentionPolicy{})
	// Added: The sample receipt exercises every persisted field in the current schema.
	want := Receipt{
		ExportVersion: "atb.custody.bundle_export.v1",
		ReceiptID:     "sha256-abc123",
		BundleHash:    "abc123",
		SubmittedAt:   time.Date(2026, time.May, 28, 0, 0, 0, 0, time.UTC),
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
	store := NewFileSystemReceiptStore(baseDir, RetentionPolicy{})

	if _, err := store.Get(context.Background(), "missing"); !errors.Is(err, ErrReceiptNotFound) {
		t.Fatalf("expected ErrReceiptNotFound, got %v", err)
	}
}

func TestReceiptStoreList(t *testing.T) {
	t.Parallel()
	// Added: A temporary directory keeps list ordering assertions deterministic.
	baseDir := t.TempDir()
	// Added: The filesystem store lists every valid receipt JSON file.
	store := NewFileSystemReceiptStore(baseDir, RetentionPolicy{})
	// Added: Distinct SubmittedAt values lock the expected ascending order.
	receipts := []Receipt{
		{
			ReceiptID:   "sha256-third",
			BundleHash:  "third",
			SubmittedAt: time.Date(2026, time.May, 28, 3, 0, 0, 0, time.UTC),
		},
		{
			ReceiptID:   "sha256-first",
			BundleHash:  "first",
			SubmittedAt: time.Date(2026, time.May, 28, 1, 0, 0, 0, time.UTC),
		},
		{
			ReceiptID:   "sha256-second",
			BundleHash:  "second",
			SubmittedAt: time.Date(2026, time.May, 28, 2, 0, 0, 0, time.UTC),
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

func TestReceiptStoreCleanUpMaxAge(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	// Policy to keep receipts for 1 day
	store := NewFileSystemReceiptStore(baseDir, RetentionPolicy{MaxAgeDays: 1})

	// Receipt from 2 days ago (should be deleted)
	oldReceipt := Receipt{
		ReceiptID:   "sha256-old",
		BundleHash:  "old",
		SubmittedAt: time.Now().Add(-48 * time.Hour),
	}
	if err := store.Save(context.Background(), oldReceipt); err != nil {
		t.Fatalf("Save oldReceipt: %v", err)
	}

	// Receipt from 1 hour ago (should be kept)
	recentReceipt := Receipt{
		ReceiptID:   "sha256-recent",
		BundleHash:  "recent",
		SubmittedAt: time.Now().Add(-1 * time.Hour),
	}
	if err := store.Save(context.Background(), recentReceipt); err != nil {
		t.Fatalf("Save recentReceipt: %v", err)
	}

	if err := store.CleanUp(context.Background()); err != nil {
		t.Fatalf("CleanUp: %v", err)
	}

	// Verify only recentReceipt remains
	receipts, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List after cleanup: %v", err)
	}
	if len(receipts) != 1 {
		t.Fatalf("expected 1 receipt after cleanup, got %d", len(receipts))
	}
	if receipts[0].ReceiptID != recentReceipt.ReceiptID {
		t.Fatalf("expected receipt %s, got %s", recentReceipt.ReceiptID, receipts[0].ReceiptID)
	}
}

func TestReceiptStoreCleanUpMaxCount(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	// Policy to keep a maximum of 2 receipts
	store := NewFileSystemReceiptStore(baseDir, RetentionPolicy{MaxCount: 2})

	// Create 3 receipts, oldest first
	receipt1 := Receipt{ReceiptID: "sha256-1", BundleHash: "1", SubmittedAt: time.Now().Add(-3 * time.Hour)}
	receipt2 := Receipt{ReceiptID: "sha256-2", BundleHash: "2", SubmittedAt: time.Now().Add(-2 * time.Hour)}
	receipt3 := Receipt{ReceiptID: "sha256-3", BundleHash: "3", SubmittedAt: time.Now().Add(-1 * time.Hour)}

	if err := store.Save(context.Background(), receipt1); err != nil {
		t.Fatalf("Save receipt1: %v", err)
	}
	if err := store.Save(context.Background(), receipt2); err != nil {
		t.Fatalf("Save receipt2: %v", err)
	}
	if err := store.Save(context.Background(), receipt3); err != nil {
		t.Fatalf("Save receipt3: %v", err)
	}

	if err := store.CleanUp(context.Background()); err != nil {
		t.Fatalf("CleanUp: %v", err)
	}

	// Verify only receipt2 and receipt3 remain
	receipts, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List after cleanup: %v", err)
	}
	if len(receipts) != 2 {
		t.Fatalf("expected 2 receipts after cleanup, got %d", len(receipts))
	}
	if receipts[0].ReceiptID != receipt2.ReceiptID || receipts[1].ReceiptID != receipt3.ReceiptID {
		t.Fatalf("expected receipts %s, %s, got %s, %s", receipt2.ReceiptID, receipt3.ReceiptID, receipts[0].ReceiptID, receipts[1].ReceiptID)
	}
}
