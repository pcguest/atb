// SPDX-License-Identifier: MIT
package proxy_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/proxy"
)

// TestSecretCanaryNeverEntersEventOrBundle injects a synthetic credential and
// asserts that neither the full value nor its last four characters reach the
// canonical event or the durable bundle bytes. This is the capture-path secret
// boundary gate for the request actor fallback.
func TestSecretCanaryNeverEntersEventOrBundle(t *testing.T) {
	const canary = "sk-CANARY9f3a"
	const suffix = "9f3a"

	rec := proxy.RequestRecord{
		SessionID:  "session-1",
		Host:       "api.openai.com",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		APIKey:     canary,
		RecordedAt: time.Now().UTC(),
	}
	ev, err := rec.ToEvent()
	if err != nil {
		t.Fatalf("ToEvent: %v", err)
	}

	rawEvent, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if strings.Contains(string(rawEvent), canary) {
		t.Fatalf("canonical event contains the full credential")
	}
	if strings.Contains(string(rawEvent), suffix) {
		t.Fatalf("canonical event contains the credential suffix: %s", rawEvent)
	}

	b, err := bundle.NewWithOptions(bundle.NewOptions{ManifestVersion: bundle.ManifestVersionV3})
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	if err := b.AppendWithOptions(ev.Type, ev.Data, &bundle.AppendOptions{Timestamp: ev.Timestamp}); err != nil {
		t.Fatalf("append: %v", err)
	}
	path := filepath.Join(t.TempDir(), "bundle.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	durable, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	// Only the full value is scanned here: the bundle is hash-chained, so a
	// short suffix like "9f3a" can legitimately appear inside a 64-hex-char
	// record hash by chance. The deterministic suffix assertion is made against
	// the canonical event above, which embeds the actor field verbatim.
	if strings.Contains(string(durable), canary) {
		t.Fatalf("durable bundle contains the full credential")
	}
}
