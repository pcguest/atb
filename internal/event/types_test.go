package event_test

import (
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// TestRegistryNotEmpty verifies the registry is populated.
func TestRegistryNotEmpty(t *testing.T) {
	if len(event.Registry) == 0 {
		t.Fatal("event.Registry must not be empty")
	}
}

// TestRegistryNoDuplicateTypes verifies no two entries share the same Type.
func TestRegistryNoDuplicateTypes(t *testing.T) {
	seen := make(map[string]bool)
	for _, entry := range event.Registry {
		if seen[entry.Type] {
			t.Errorf("duplicate event type in Registry: %q", entry.Type)
		}
		seen[entry.Type] = true
	}
}

// TestBundleConstantsMatch guards cross-package consistency.
func TestBundleConstantsMatch(t *testing.T) {
	if event.TypeBundleManifest != bundle.ManifestEventType {
		t.Errorf(
			"TypeBundleManifest %q != bundle.ManifestEventType %q",
			event.TypeBundleManifest,
			bundle.ManifestEventType,
		)
	}
	if event.TypeBundleAnchor != bundle.AnchorEventType {
		t.Errorf(
			"TypeBundleAnchor %q != bundle.AnchorEventType %q",
			event.TypeBundleAnchor,
			bundle.AnchorEventType,
		)
	}
}

// TestAllCriticalProfileEventsPresent checks required event types are registered.
func TestAllCriticalProfileEventsPresent(t *testing.T) {
	required := []string{
		event.TypeAIActionPrecommit,
		event.TypeAIActionExecuted,
		event.TypeAIActionCommitted,
		event.TypeAIModelInvoked,
		event.TypeAIModelOutput,
		event.TypeAIRequestReceived,
	}

	typeSet := make(map[string]bool, len(event.Registry))
	for _, entry := range event.Registry {
		typeSet[entry.Type] = true
	}
	for _, requiredType := range required {
		if !typeSet[requiredType] {
			t.Errorf("required event type missing from Registry: %q", requiredType)
		}
	}
}
