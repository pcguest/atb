// SPDX-License-Identifier: MIT
package capabilities

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

func TestDeriveRetrievalCapabilityPreservesRawIdentity(t *testing.T) {
	t.Parallel()

	records := []bundle.Record{
		{Event: hash.Event{Sequence: 2, Type: "atb.event.rag_retrieval", Data: map[string]any{
			"retrieval_id":      "ret-1",
			"index_id":          "idx-1",
			"result_set_digest": "abc",
			"query":             "must not be copied",
		}}},
	}

	got := Derive(records)
	if len(got) != 1 {
		t.Fatalf("Derive() returned %d capabilities, want 1", len(got))
	}
	if got[0].Name != "retrieval.performed" || got[0].RawEventType != "atb.event.rag_retrieval" {
		t.Fatalf("capability = %#v", got[0])
	}
	if _, copiedRawQuery := got[0].Fields["query"]; copiedRawQuery {
		t.Fatal("canonical capability must not copy unbounded raw query content")
	}
}
