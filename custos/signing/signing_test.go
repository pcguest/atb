// SPDX-License-Identifier: MIT
package signing_test

import (
	"testing"

	"github.com/pcguest/custos/signing"
)

func TestSigningPolicy_zeroValueConstructible(t *testing.T) {
	t.Parallel()

	var policy signing.SigningPolicy

	if policy.OrgID != "" {
		t.Fatalf("OrgID = %q, want empty string", policy.OrgID)
	}
	if policy.KeySource != signing.KeySourceLocalFile {
		t.Fatalf("KeySource = %d, want %d", policy.KeySource, signing.KeySourceLocalFile)
	}
}

func TestBundleSigningResult_unsignedZeroState(t *testing.T) {
	t.Parallel()

	result := signing.BundleSigningResult{
		Signed:      false,
		TSAAnchored: false,
	}

	if result.Signed {
		t.Fatal("Signed = true, want false")
	}
	if result.TSAAnchored {
		t.Fatal("TSAAnchored = true, want false")
	}
}

func TestKeySource_constants(t *testing.T) {
	t.Parallel()

	if signing.KeySourceLocalFile != 0 {
		t.Fatalf("KeySourceLocalFile = %d, want 0", signing.KeySourceLocalFile)
	}
	if signing.KeySourceKMS != 1 {
		t.Fatalf("KeySourceKMS = %d, want 1", signing.KeySourceKMS)
	}
}
