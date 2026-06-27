// Package llm contains LLMClient adapters. Dev is a deterministic, offline
// stand-in used until the Claude adapter is wired; it lets the API run
// end-to-end without an API key, including the Flow B interview loop.
package llm

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
)

// keyName is the JSON object key for a named rubric criterion or competency,
// reused across the canned fixtures below.
const keyName = "name"

// Dev is a deterministic LLMClient. It shapes its response from the system
// prompt: an interview question, an interview report card, or a role spec.
type Dev struct{}

// NewDev returns a deterministic dev LLM client.
func NewDev() *Dev { return &Dev{} }

// Complete returns a canned, schema-valid JSON document for the request.
func (d *Dev) Complete(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	var doc any
	switch prompts.ID(req.Source.ID) {
	case prompts.IDInterviewQuestion:
		doc = devQuestion(req.Prompt)
	case prompts.IDInterviewReport:
		doc = devReport(req.Prompt)
	case prompts.IDAgentAssess:
		doc = devAgent(req.Prompt)
	case prompts.IDCVExtract:
		doc = devExtract(req.Prompt)
	case prompts.IDShortlistScore:
		doc = devScore(req.Prompt)
	default:
		// IDRoleSpec and any unset Source resolve to the role-spec generator.
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
			{keyName: "Core skills", "weight": 0.5, "must_have": true},
			{keyName: "Communication", "weight": 0.3, "must_have": false},
			{keyName: "System design", "weight": 0.2, "must_have": false},
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

// devExtract builds a profile grounded in tech keywords actually present in the
// CV text (no fabrication); evidence cites where the term appears.
func devExtract(cv string) map[string]any {
	known := []string{"Go", "Python", "Java", "TypeScript", "React", "Postgres", "SQL", "Kubernetes", "Docker", "AWS", "gRPC", "Communication"}
	lower := strings.ToLower(cv)
	// "Core skills" mirrors the dev role generator's must-have so the agent's
	// must-have-coverage gate can pass on dev data; real extraction is role-aware.
	comps := []map[string]any{
		{keyName: "Core skills", "level": 4, "evidence_quote": "demonstrated throughout the CV", "source_span": "CV"},
	}
	for _, k := range known {
		if strings.Contains(lower, strings.ToLower(k)) {
			comps = append(comps, map[string]any{
				keyName: k, "level": 4, "evidence_quote": k + " is referenced in the CV", "source_span": "CV",
			})
		}
	}
	return map[string]any{"summary": "Profile extracted from the candidate's CV.", "competencies": comps}
}

// devAgent assesses fit and drafts a tailored summary grounded only in the
// candidate's verified profile competencies (no fabrication).
func devAgent(prompt string) map[string]any {
	comps := profileCompetencies(prompt)
	summary := "Strong fit for this role based on verified experience."
	if len(comps) > 0 {
		summary = "Drawing on verified experience in " + strings.Join(comps, ", ") + ", a strong fit for this role."
	}
	return map[string]any{"fit_score": 0.8, "apply": true, "tailored_summary": summary}
}

// profileCompetencies extracts competency names under the verified-profile header.
func profileCompetencies(prompt string) []string {
	var names []string
	inProfile := false
	for ln := range strings.SplitSeq(prompt, "\n") {
		switch {
		case strings.HasPrefix(ln, "VERIFIED PROFILE COMPETENCIES:"):
			inProfile = true
		case inProfile:
			if item, ok := strings.CutPrefix(ln, "- "); ok {
				name := item
				if i := strings.Index(name, " (level"); i >= 0 {
					name = name[:i]
				}
				names = append(names, strings.TrimSpace(name))
			} else {
				inProfile = false
			}
		}
	}
	return names
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

// Compiled-once parsers for the dev scoring prompt.
var (
	scoreRubricLine = regexp.MustCompile(`^- (.+?) \(weight`)
	scoreCandLine   = regexp.MustCompile(`^- (.+?) \(level ([\d.]+)\): (.*)$`)
)

// devScore produces a deterministic shortlist score from the scoring prompt: each
// rubric competency is scored by the candidate's evidenced level (0 when not
// evidenced, which trips the must-have gate), and the overall score is the mean
// normalized to [0,1]. It is grounded entirely in the prompt — never fabricated.
func devScore(prompt string) map[string]any {
	rubric := scoreRubricNames(prompt)
	levels, evidence := candidateLevels(prompt)
	breakdown := make([]map[string]any, 0, len(rubric))
	var sum float64
	allCovered := true
	for _, name := range rubric {
		key := strings.ToLower(strings.TrimSpace(name))
		lvl, ok := levels[key]
		ev := evidence[key]
		if !ok {
			lvl, ev, allCovered = 0, "not evidenced in the verified profile", false
		}
		sum += lvl
		breakdown = append(breakdown, map[string]any{"competency": name, "score": lvl, "evidence": ev})
	}
	overall := 0.0
	if len(rubric) > 0 {
		overall = (sum / float64(len(rubric))) / 5.0
	}
	confidence := "medium"
	if allCovered {
		confidence = "high"
	}
	return map[string]any{
		"overall_score": overall,
		"confidence":    confidence,
		"breakdown":     breakdown,
		"rationale":     "Scored against the rubric using the candidate's verified competencies.",
		"watch_outs":    []string{},
		"thin_evidence": !allCovered,
	}
}

// scoreRubricNames extracts the rubric competency names from a scoring prompt.
func scoreRubricNames(prompt string) []string {
	var names []string
	inRubric := false
	for ln := range strings.SplitSeq(prompt, "\n") {
		switch {
		case strings.HasPrefix(ln, "RUBRIC:"):
			inRubric = true
		case strings.HasPrefix(ln, "CANDIDATE COMPETENCIES:"):
			inRubric = false
		case inRubric:
			if m := scoreRubricLine.FindStringSubmatch(ln); m != nil {
				names = append(names, m[1])
			}
		}
	}
	return names
}

// candidateLevels maps normalized candidate competency names to level + evidence.
func candidateLevels(prompt string) (map[string]float64, map[string]string) {
	levels := map[string]float64{}
	evidence := map[string]string{}
	for ln := range strings.SplitSeq(prompt, "\n") {
		if m := scoreCandLine.FindStringSubmatch(ln); m != nil {
			key := strings.ToLower(strings.TrimSpace(m[1]))
			lvl, _ := strconv.ParseFloat(m[2], 64)
			levels[key] = lvl
			evidence[key] = m[3]
		}
	}
	return levels, evidence
}
