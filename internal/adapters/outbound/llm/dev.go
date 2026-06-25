// Package llm contains LLMClient adapters. Dev is a deterministic, offline
// stand-in used until the Claude adapter (CAL-030) is wired; it lets the API
// run end-to-end without an API key.
package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
)

// Dev is a deterministic LLMClient that returns a plausible role-spec JSON.
type Dev struct{}

// NewDev returns a deterministic dev LLM client.
func NewDev() *Dev { return &Dev{} }

// Complete returns a canned, schema-valid role spec derived from the prompt.
func (d *Dev) Complete(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	title := firstLine(req.Prompt)
	if title == "" {
		title = "Software Engineer"
	}
	doc := map[string]any{
		"title":            title,
		"location":         "Accra, Ghana",
		"seniority":        "mid",
		"availability":     "within 1 month",
		"responsibilities": []string{"Build and maintain services", "Collaborate with the team"},
		"must_haves":       []string{"3+ years experience"},
		"nice_to_haves":    []string{"Domain knowledge"},
		"salary_band":      map[string]any{"currency": "GHS", "low": 0, "high": 0},
		"rubric": []map[string]any{
			{"name": "Core skills", "weight": 0.5, "must_have": true},
			{"name": "Communication", "weight": 0.3, "must_have": false},
			{"name": "System design", "weight": 0.2, "must_have": false},
		},
	}
	b, _ := json.Marshal(doc)
	return app.LLMResponse{Text: string(b)}, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

var _ app.LLMClient = (*Dev)(nil)
