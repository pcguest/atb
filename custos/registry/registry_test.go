// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pcguest/custos/internal/receipt"
)

func rec(id, hash, submittedAt string) receipt.Receipt {
	t, _ := time.Parse(time.RFC3339, submittedAt)
	return receipt.Receipt{
		ExportVersion: "custos.receipt.v1",
		ReceiptID:     id,
		BundleHash:    hash,
		SubmittedAt:   t,
	}
}

func TestRegisterAndGetByReceiptID(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()

	in := rec("sha256-aaa", "head-1", "2026-06-05T10:00:00Z")
	if err := r.Register(ctx, in); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := r.GetByReceiptID(ctx, "sha256-aaa")
	if err != nil {
		t.Fatalf("GetByReceiptID: %v", err)
	}
	if got.ReceiptID != in.ReceiptID || got.BundleHash != in.BundleHash {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, in)
	}
}

func TestGetByReceiptIDMissing(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()
	if _, err := r.GetByReceiptID(ctx, "sha256-missing"); !errors.Is(err, ErrReceiptNotFound) {
		t.Fatalf("err = %v, want ErrReceiptNotFound", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()

	if err := r.Register(ctx, rec("", "head-1", "t")); err == nil {
		t.Fatal("Register with empty receipt id: want error")
	}
	if err := r.Register(ctx, rec("sha256-aaa", "  ", "t")); err == nil {
		t.Fatal("Register with empty bundle hash: want error")
	}
}

func TestFindByBundleHashReturnsAllMatches(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()

	// Two distinct receipts (different bytes) custodying the same chain-head hash.
	if err := r.Register(ctx, rec("sha256-bbb", "head-shared", "2026-06-05T11:00:00Z")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := r.Register(ctx, rec("sha256-aaa", "head-shared", "2026-06-05T10:00:00Z")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := r.Register(ctx, rec("sha256-ccc", "head-other", "2026-06-05T12:00:00Z")); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := r.FindByBundleHash(ctx, "head-shared")
	if err != nil {
		t.Fatalf("FindByBundleHash: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d receipts, want 2", len(got))
	}
	// Sorted by SubmittedAt ascending: aaa (10:00) before bbb (11:00).
	if got[0].ReceiptID != "sha256-aaa" || got[1].ReceiptID != "sha256-bbb" {
		t.Fatalf("unexpected order: %s, %s", got[0].ReceiptID, got[1].ReceiptID)
	}
}

func TestFindByBundleHashNoMatchIsEmptyNotError(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()
	if err := r.Register(ctx, rec("sha256-aaa", "head-1", "t")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, err := r.FindByBundleHash(ctx, "head-unknown")
	if err != nil {
		t.Fatalf("FindByBundleHash: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d receipts, want 0", len(got))
	}
}

func TestRegisterIsIdempotentUpsert(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()

	first := rec("sha256-aaa", "head-1", "2026-06-05T10:00:00Z")
	first.SubmitterRef = "alice"
	if err := r.Register(ctx, first); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Re-register the same receipt ID with updated metadata.
	second := rec("sha256-aaa", "head-1", "2026-06-05T10:00:00Z")
	second.SubmitterRef = "bob"
	if err := r.Register(ctx, second); err != nil {
		t.Fatalf("re-Register: %v", err)
	}

	all, err := r.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("idempotent upsert produced %d entries, want 1", len(all))
	}
	if all[0].SubmitterRef != "bob" {
		t.Fatalf("upsert did not refresh entry: got %q, want bob", all[0].SubmitterRef)
	}
}

func TestRegisterReindexesOnBundleHashChange(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()

	if err := r.Register(ctx, rec("sha256-aaa", "head-old", "t")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Same receipt ID, different bundle hash (defensive: should not normally
	// happen for a content-addressed ID, but the index must stay consistent).
	if err := r.Register(ctx, rec("sha256-aaa", "head-new", "t")); err != nil {
		t.Fatalf("re-Register: %v", err)
	}

	old, err := r.FindByBundleHash(ctx, "head-old")
	if err != nil {
		t.Fatalf("FindByBundleHash(old): %v", err)
	}
	if len(old) != 0 {
		t.Fatalf("stale digest mapping survived: got %d, want 0", len(old))
	}
	cur, err := r.FindByBundleHash(ctx, "head-new")
	if err != nil {
		t.Fatalf("FindByBundleHash(new): %v", err)
	}
	if len(cur) != 1 {
		t.Fatalf("new digest mapping missing: got %d, want 1", len(cur))
	}
}

func TestListSortedBySubmittedAtThenReceiptID(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryRegistry()

	_ = r.Register(ctx, rec("sha256-z", "h1", "2026-06-05T12:00:00Z"))
	_ = r.Register(ctx, rec("sha256-a", "h2", "2026-06-05T10:00:00Z"))
	_ = r.Register(ctx, rec("sha256-b", "h3", "2026-06-05T10:00:00Z")) // tie on time, break by ID

	all, err := r.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{"sha256-a", "sha256-b", "sha256-z"}
	for i, w := range want {
		if all[i].ReceiptID != w {
			t.Fatalf("List order[%d] = %s, want %s", i, all[i].ReceiptID, w)
		}
	}
}

func TestBuildFromReceiptStore(t *testing.T) {
	ctx := context.Background()
	store := receipt.NewInMemoryReceiptStore()
	if err := store.Save(ctx, rec("sha256-aaa", "head-1", "2026-06-05T10:00:00Z")); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	if err := store.Save(ctx, rec("sha256-bbb", "head-1", "2026-06-05T11:00:00Z")); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	r, err := Build(ctx, store)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got, err := r.FindByBundleHash(ctx, "head-1")
	if err != nil {
		t.Fatalf("FindByBundleHash: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Build indexed %d receipts for head-1, want 2", len(got))
	}
}

func TestContextCancellationIsHonoured(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := NewInMemoryRegistry()

	if err := r.Register(ctx, rec("sha256-aaa", "head-1", "t")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Register err = %v, want context.Canceled", err)
	}
	if _, err := r.GetByReceiptID(ctx, "sha256-aaa"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByReceiptID err = %v, want context.Canceled", err)
	}
	if _, err := r.FindByBundleHash(ctx, "head-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("FindByBundleHash err = %v, want context.Canceled", err)
	}
	if _, err := r.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("List err = %v, want context.Canceled", err)
	}
}
