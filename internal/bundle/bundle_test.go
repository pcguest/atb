package bundle_test

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestAppendRejectsInvalidEventType(t *testing.T) {
	b := bundle.New()

	if err := b.Append("INVALID", nil); err == nil {
		t.Fatalf("expected invalid event type to return an error")
	}
}
