package viewer

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

//go:embed template.html
var templateHTML string

var pageTemplate = template.Must(template.New("timeline").Parse(templateHTML))

// EventCard is the normalized event representation used by the viewer template.
type EventCard struct {
	Sequence  int
	Type      string
	Timestamp string
	Actor     string
	Preview   string
	DataJSON  string
	Hash      string
	PrevHash  string
}

// PageData contains everything required to render the timeline HTML page.
type PageData struct {
	BundlePath   string
	GeneratedAt  string
	VerifyOK     bool
	VerifyStatus string
	GateStatus   string
	Events       []EventCard
}

// BuildPageData prepares render-ready data from a loaded bundle.
func BuildPageData(b *bundle.Bundle, bundlePath string) PageData {
	verifyErr := b.Verify()
	status := "✓ Hash chain verified"
	verifyOK := true
	if verifyErr != nil {
		status = fmt.Sprintf("✗ Verification failed: %v", verifyErr)
		verifyOK = false
	}

	cards := make([]EventCard, 0, len(b.Records))
	for _, r := range b.Records {
		timestamp, actor := extractMetadata(r.Event.Data)
		cards = append(cards, EventCard{
			Sequence:  r.Event.Sequence,
			Type:      r.Event.Type,
			Timestamp: timestamp,
			Actor:     actor,
			Preview:   summarizeData(r.Event.Data),
			DataJSON:  prettyJSON(r.Event.Data),
			Hash:      r.Hash,
			PrevHash:  r.Event.PrevHash,
		})
	}

	return PageData{
		BundlePath:   bundlePath,
		GeneratedAt:  time.Now().Format(time.RFC3339),
		VerifyOK:     verifyOK,
		VerifyStatus: status,
		GateStatus:   detectGateStatus(b.Records),
		Events:       cards,
	}
}

// NewHandler returns a handler that serves a static timeline page for the
// provided page data.
func NewHandler(page PageData) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pageTemplate.Execute(w, page); err != nil {
			http.Error(w, fmt.Sprintf("render timeline: %v", err), http.StatusInternalServerError)
		}
	})
}

func detectGateStatus(records []bundle.Record) string {
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

func extractMetadata(data interface{}) (string, string) {
	m, ok := data.(map[string]interface{})
	if !ok {
		return "-", "-"
	}

	timestampKeys := []string{"timestamp", "time", "created_at", "ts", "date"}
	actorKeys := []string{"actor", "agent", "user", "author"}
	return firstString(m, timestampKeys, "-"), firstString(m, actorKeys, "-")
}

func firstString(m map[string]interface{}, keys []string, fallback string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return fallback
}

func summarizeData(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		encoded, err := json.Marshal(data)
		if err != nil {
			return "(unavailable)"
		}
		return truncate(string(encoded), 160)
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
	return truncate(strings.Join(parts, ", "), 160)
}

func prettyJSON(data interface{}) string {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "(unable to render JSON)"
	}
	return string(encoded)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}
