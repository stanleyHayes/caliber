// Package llm contains LLMClient adapters. Dev is a deterministic, offline
// stand-in used until the Claude adapter is wired; it lets the API run
// end-to-end without an API key, including the Flow B interview loop.
package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
)

// Dev is a deterministic LLMClient. It shapes its response from the system
// prompt: an interview question, an interview report card, or a role spec.
type Dev struct{}

// NewDev returns a deterministic dev LLM client.
func NewDev() *Dev { return &Dev{} }

// Complete returns a canned, schema-valid JSON document for the request.
func (d *Dev) Complete(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	var doc any
	switch {
	case strings.Contains(req.System, "screening interviewer"):
		doc = devQuestion(req.Prompt)
	case strings.Contains(req.System, "score a screening interview"):
		doc = devReport(req.Prompt)
	default:
		doc = devRoleSpec(req.Prompt)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return app.LLMResponse{}, err
	}
	return app.LLMResponse{Text: string(b)}, nil
}

func devRoleSpec(prompt string) map[string]any {
	title := firstLine(prompt)
	if title == "" {
		title = "Software Engineer"
	}
	return map[string]any{
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
}

func devQuestion(prompt string) map[string]any {
	names := rubricNames(prompt)
	comp := names[len(answers(prompt))%len(names)]
	return map[string]any{
		"question":       "Walk me through a concrete example of your " + comp + " work, with the trade-offs you made.",
		"competency_tag": comp,
	}
}

func devReport(prompt string) map[string]any {
	names := rubricNames(prompt)
	ans := answers(prompt)
	evidence := "no concrete example was provided"
	if len(ans) > 0 {
		evidence = ans[0] // quote the candidate's own words (no fabrication)
	}
	scores := make([]map[string]any, 0, len(names))
	for _, n := range names {
		scores = append(scores, map[string]any{"competency": n, "score": 3.5, "evidence": evidence})
	}
	return map[string]any{
		"verdict":               "advance",
		"confidence":            "medium",
		"scores":                scores,
		"recommended_next_step": "Proceed to a technical onsite.",
	}
}

// rubricNames extracts the "- name" lines under a "RUBRIC:" header in the prompt.
func rubricNames(prompt string) []string {
	var names []string
	inRubric := false
	for ln := range strings.SplitSeq(prompt, "\n") {
		switch {
		case strings.HasPrefix(ln, "RUBRIC:"):
			inRubric = true
		case inRubric:
			if name, ok := strings.CutPrefix(ln, "- "); ok {
				names = append(names, strings.TrimSpace(name))
			} else {
				inRubric = false
			}
		}
	}
	if len(names) == 0 {
		return []string{"Core skills"}
	}
	return names
}

// answers extracts the candidate's "A: " transcript lines from the prompt.
func answers(prompt string) []string {
	var out []string
	for ln := range strings.SplitSeq(prompt, "\n") {
		if a, ok := strings.CutPrefix(ln, "A: "); ok {
			out = append(out, strings.TrimSpace(a))
		}
	}
	return out
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
