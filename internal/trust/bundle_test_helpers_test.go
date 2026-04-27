// SPDX-License-Identifier: MIT
package trust

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func newTrustTestBundle(t testing.TB) *bundle.Bundle {
	t.Helper()

	b, err := bundle.New()
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	return b
}
