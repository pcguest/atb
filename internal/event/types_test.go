package event_test

import (
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/profiles"
	"github.com/pcguest/atb/internal/verify"
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

func TestRegistryProfileAnnotationsCoverEmbeddedTemplates(t *testing.T) {
	registryByType := make(map[string]event.EventInfo, len(event.Registry))
	for _, entry := range event.Registry {
		registryByType[entry.Type] = entry
	}

	for _, profile := range verify.AllProfiles() {
		schema := profiles.MustLoadSchema(profile.ID())
		rules := make([]profiles.EventRule, 0, len(schema.Required)+len(schema.Optional))
		rules = append(rules, schema.Required...)
		rules = append(rules, schema.Optional...)

		for _, rule := range rules {
			entry, ok := registryByType[rule.Type]
			if !ok {
				t.Fatalf("registry missing event type %q for profile %q", rule.Type, profile.ID())
			}
			if !profileListed(entry.Profile, profile.ID()) {
				t.Errorf("registry entry %q missing profile annotation %q", rule.Type, profile.ID())
			}
		}
	}
}

func profileListed(profileList string, profileID string) bool {
	for _, candidate := range strings.Split(profileList, ",") {
		if strings.TrimSpace(candidate) == profileID {
			return true
		}
	}
	return false
}
