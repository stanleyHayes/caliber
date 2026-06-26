// Package prompts is the versioned prompt registry (CAL-032). Every system
// prompt the platform sends lives here as a VCS file under files/<id>/<version>,
// compiled in via go:embed and referenced by a typed id. Building a request
// through a Prompt stamps its id + version onto the LLMRequest, so the audit
// trail records exactly which prompt produced each model call — replacing the
// fragile substring classification it used before.
package prompts

import (
	"embed"
	"fmt"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
)

// ID is a stable, typed prompt identifier. The values match the operation labels
// the audit feed has always used, for dashboard/log continuity.
type ID string

// Registered prompt ids.
const (
	IDInterviewQuestion ID = "interview_question"
	IDInterviewReport   ID = "interview_report"
	IDAgentAssess       ID = "agent_assess"
	IDCVExtract         ID = "cv_extract"
	IDShortlistScore    ID = "shortlist_score"
	IDRoleSpec          ID = "role_spec"
)

// Prompt is a registered, versioned system prompt with its token budget.
type Prompt struct {
	ID        ID
	Version   string
	Body      string
	MaxTokens int
}

// Ref returns the audit reference (id + version) for the prompt.
func (p Prompt) Ref() app.PromptRef { return app.PromptRef{ID: string(p.ID), Version: p.Version} }

// Request builds a fully-formed LLMRequest from the prompt, stamping its id and
// version onto the request for traceability. userPrompt is the (already-fenced,
// where applicable) untrusted content that becomes the request body.
func (p Prompt) Request(userPrompt string) app.LLMRequest {
	return app.LLMRequest{System: p.Body, Prompt: userPrompt, MaxTokens: p.MaxTokens, Source: p.Ref()}
}

//go:embed files
var files embed.FS

// reg is one registration: the single source of truth tying id, version, file,
// and token budget together.
type reg struct {
	id        ID
	version   string
	path      string
	maxTokens int
}

//nolint:gochecknoglobals // immutable registry built once at init from embedded files
var registry = mustLoad([]reg{
	{IDInterviewQuestion, "v1", "files/interview_question/v1.txt", 512},
	{IDInterviewReport, "v1", "files/interview_report/v1.txt", 1024},
	{IDAgentAssess, "v1", "files/agent_assess/v1.txt", 768},
	{IDCVExtract, "v1", "files/cv_extract/v1.txt", 1024},
	{IDShortlistScore, "v1", "files/shortlist_score/v1.txt", 1024},
	{IDRoleSpec, "v1", "files/role_spec/v1.txt", 1024},
})

// mustLoad reads every registered prompt from the embedded corpus at init,
// failing fast (panic) on a duplicate id, missing file, or empty body so a
// malformed registry can never ship in a running binary.
func mustLoad(entries []reg) map[ID]Prompt {
	out := make(map[ID]Prompt, len(entries))
	for _, e := range entries {
		if _, dup := out[e.id]; dup {
			panic(fmt.Sprintf("prompts: duplicate id %q", e.id))
		}
		body, err := files.ReadFile(e.path)
		if err != nil {
			panic(fmt.Sprintf("prompts: cannot load %q: %v", e.path, err))
		}
		if strings.TrimSpace(string(body)) == "" {
			panic(fmt.Sprintf("prompts: empty body for %q", e.id))
		}
		out[e.id] = Prompt{ID: e.id, Version: e.version, Body: string(body), MaxTokens: e.maxTokens}
	}
	return out
}

// Get returns the registered prompt for id, panicking on an unknown id — a
// programmer error caught at init/test time, never reached with a valid const.
func Get(id ID) Prompt {
	p, ok := registry[id]
	if !ok {
		panic(fmt.Sprintf("prompts: unknown id %q", id))
	}
	return p
}

// All returns every registered prompt (order unspecified).
func All() []Prompt {
	out := make([]Prompt, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}
