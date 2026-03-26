package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

//go:embed view_legacy_template.html
var legacyViewTemplateHTML string

var legacyViewTemplate = template.Must(template.New("timeline").Parse(legacyViewTemplateHTML))

type legacyEventCard struct {
	Sequence  int
	Type      string
	Timestamp string
	Actor     string
	Preview   string
	DataJSON  string
	Hash      string
	PrevHash  string
}

type legacyPageData struct {
	BundlePath   string
	GeneratedAt  string
	VerifyOK     bool
	VerifyStatus string
	GateStatus   string
	Events       []legacyEventCard
}

func buildLegacyPageData(b *bundle.Bundle, bundlePath string) legacyPageData {
	verifyErr := b.Verify()
	status := "✓ Hash chain verified"
	verifyOK := true
	if verifyErr != nil {
		status = fmt.Sprintf("✗ Verification failed: %v", verifyErr)
		verifyOK = false
	}

	cards := make([]legacyEventCard, 0, len(b.Records))
	for _, r := range b.Records {
		timestamp, actor := extractLegacyMetadata(r.Event.Data)
		cards = append(cards, legacyEventCard{
			Sequence:  r.Event.Sequence,
			Type:      r.Event.Type,
			Timestamp: timestamp,
			Actor:     actor,
			Preview:   summarizeLegacyData(r.Event.Data),
			DataJSON:  prettyLegacyJSON(r.Event.Data),
			Hash:      r.Hash,
			PrevHash:  r.Event.PrevHash,
		})
	}

	return legacyPageData{
		BundlePath:   bundlePath,
		GeneratedAt:  time.Now().Format(time.RFC3339),
		VerifyOK:     verifyOK,
		VerifyStatus: status,
		GateStatus:   detectLegacyGateStatus(b.Records),
		Events:       cards,
	}
}

func newLegacyViewHandler(page legacyPageData) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := legacyViewTemplate.Execute(w, page); err != nil {
			http.Error(w, fmt.Sprintf("render timeline: %v", err), http.StatusInternalServerError)
		}
	})
}

func detectLegacyGateStatus(records []bundle.Record) string {
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if !strings.Contains(r.Event.Type, "snapshot") {
			continue
		}
		if data, ok := r.Event.Data.(map[string]interface{}); ok {
			if gate, ok := data["gate"].(string); ok {
				gate = strings.ToUpper(strings.TrimSpace(gate))
				if gate == "PASS" || gate == "FAIL" {
					return gate
				}
			}
		}
	}
	return "UNKNOWN"
}

func extractLegacyMetadata(data interface{}) (string, string) {
	m, ok := data.(map[string]interface{})
	if !ok {
		return "-", "-"
	}

	timestampKeys := []string{"timestamp", "time", "created_at", "ts", "date"}
	actorKeys := []string{"actor", "agent", "user", "author"}
	return firstLegacyString(m, timestampKeys, "-"), firstLegacyString(m, actorKeys, "-")
}

func firstLegacyString(m map[string]interface{}, keys []string, fallback string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return fallback
}

func summarizeLegacyData(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		encoded, err := json.Marshal(data)
		if err != nil {
			return "(unavailable)"
		}
		return truncateLegacyString(string(encoded), 160)
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "{}"
	}

	parts := make([]string, 0, 3)
	for i, key := range keys {
		if i == 3 {
			parts = append(parts, "…")
			break
		}
		encoded, err := json.Marshal(m[key])
		if err != nil {
			encoded = []byte("null")
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, string(encoded)))
	}
	return truncateLegacyString(strings.Join(parts, ", "), 160)
}

func prettyLegacyJSON(data interface{}) string {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "(unable to render JSON)"
	}
	return string(encoded)
}

func truncateLegacyString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}
