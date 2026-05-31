// SPDX-License-Identifier: MIT
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

// TestRegistryMatchesGenerated guards against the legacy hand-maintained
// Registry silently diverging from the schema-generated RegistryGenerated.
// Both must describe exactly the same set of event types; the schema
// (schemas/event.v1.json) remains the single source of truth.
func TestRegistryMatchesGenerated(t *testing.T) {
	legacy := make(map[string]bool, len(event.Registry))
	for _, entry := range event.Registry {
		legacy[entry.Type] = true
	}
	generated := make(map[string]bool, len(event.RegistryGenerated))
	for _, entry := range event.RegistryGenerated {
		generated[entry.Type] = true
	}

	for typ := range generated {
		if !legacy[typ] {
			t.Errorf("event type %q is in RegistryGenerated but missing from legacy Registry", typ)
		}
	}
	for typ := range legacy {
		if !generated[typ] {
			t.Errorf("event type %q is in legacy Registry but missing from RegistryGenerated (stale entry)", typ)
		}
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

// TestActionErrorEventRegistered pins the forensic ai.action.error event:
// it must be registered with action_id + error_class as required fields, and
// must NOT be bound to any obligation profile (it is an additive forensic
// vocabulary populated by capture, not a scored obligation — keeping it out of
// profiles is what makes the addition non-breaking for existing bundles).
func TestActionErrorEventRegistered(t *testing.T) {
	if event.TypeAIActionError != "ai.action.error" {
		t.Fatalf("TypeAIActionError = %q, want ai.action.error", event.TypeAIActionError)
	}

	var spec *event.EventTypeSpecGenerated
	for i := range event.EventTypesGenerated {
		if event.EventTypesGenerated[i].Type == event.TypeAIActionError {
			spec = &event.EventTypesGenerated[i]
			break
		}
	}
	if spec == nil {
		t.Fatal("ai.action.error missing from EventTypesGenerated")
	}
	if spec.Criticality != "required" {
		t.Errorf("ai.action.error criticality = %q, want required", spec.Criticality)
	}
	if len(spec.Profiles) != 0 {
		t.Errorf("ai.action.error must not be bound to any profile, got %v", spec.Profiles)
	}

	want := map[string]bool{"action_id": true, "error_class": true}
	got := event.RequiredFieldsGenerated[event.TypeAIActionError]
	if len(got) != len(want) {
		t.Fatalf("ai.action.error required_fields = %v, want keys %v", got, want)
	}
	for _, f := range got {
		if !want[f] {
			t.Errorf("ai.action.error unexpected required field %q", f)
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
