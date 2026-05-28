// SPDX-License-Identifier: MIT
package ingestion_test

import (
	"testing"

	"github.com/pcguest/custos/ingestion"
	"github.com/pcguest/custos/signing"

	atbv1 "github.com/pcguest/atb/pkg/api/v1"
)

type testIngestor struct{}

func (testIngestor) Receives(ingestion.ToolEvent) (*atbv1.EventRecordDTO, error) {
	return nil, nil
}

var _ ingestion.Ingestor = testIngestor{}

func TestToolEvent_zeroValueConstructible(t *testing.T) {
	t.Parallel()

	var event ingestion.ToolEvent

	if event.SigningResult != nil {
		t.Fatal("SigningResult = non-nil, want nil")
	}
}

func TestToolEvent_signingResultAccessible(t *testing.T) {
	t.Parallel()

	event := ingestion.ToolEvent{
		SigningResult: &signing.BundleSigningResult{
			BundleID: "bundle-1",
			Signed:   true,
		},
	}

	if event.SigningResult == nil {
		t.Fatal("SigningResult = nil, want non-nil")
	}
	if event.SigningResult.BundleID != "bundle-1" {
		t.Fatalf("SigningResult.BundleID = %q, want bundle-1", event.SigningResult.BundleID)
	}
}

func TestIngestor_interfaceSatisfiedByStub(t *testing.T) {
	t.Parallel()

	var ingestor ingestion.Ingestor = testIngestor{}

	record, err := ingestor.Receives(ingestion.ToolEvent{})
	if err != nil {
		t.Fatalf("Receives returned error: %v", err)
	}
	if record != nil {
		t.Fatal("Receives returned non-nil record, want nil")
	}
}
