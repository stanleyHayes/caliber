package prompts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistry_EveryIDResolves(t *testing.T) {
	// every prompt id the platform relies on; each must be registered.
	allIDs := []ID{
		IDInterviewQuestion, IDInterviewReport, IDAgentAssess,
		IDCVExtract, IDShortlistScore, IDRoleSpec,
	}
	for _, id := range allIDs {
		p := Get(id)
		assert.Equalf(t, id, p.ID, "id round-trips for %s", id)
		assert.NotEmptyf(t, strings.TrimSpace(p.Body), "non-empty body for %s", id)
		assert.Equalf(t, "v1", p.Version, "version for %s", id)
		assert.Positivef(t, p.MaxTokens, "positive token budget for %s", id)
		assert.Falsef(t, strings.HasSuffix(p.Body, "\n"),
			"body for %s must not carry a trailing newline the const lacked", id)
	}
	assert.Len(t, All(), len(allIDs), "All() returns exactly the registered prompts")
}

// TestRegistry_GoldenContent guards the load-bearing content of each prompt: the
// CAL-119 data-only fence notice (where applicable) and the identity phrase that
// names the operation. A byte-drift in a migrated file that dropped these would
// fail here.
func TestRegistry_GoldenContent(t *testing.T) {
	identity := map[ID]string{
		IDInterviewQuestion: "adaptive technical screening interviewer",
		IDInterviewReport:   "score a screening interview",
		IDAgentAssess:       "honest job-application agent",
		IDCVExtract:         "structured talent profile from a CV",
		IDShortlistScore:    "candidate against a role rubric",
		IDRoleSpec:          "structured role spec",
	}
	for id, phrase := range identity {
		assert.Containsf(t, Get(id).Body, phrase, "%s keeps its identity phrase", id)
	}
	// Prompts that consume fenced untrusted content must keep the data-only notice.
	for _, id := range []ID{IDInterviewQuestion, IDInterviewReport, IDCVExtract, IDShortlistScore, IDAgentAssess, IDRoleSpec} {
		assert.Containsf(t, Get(id).Body, "UNTRUSTED",
			"%s keeps its CAL-119 data-only fence notice", id)
	}
	// The scoring prompt must keep the explicit protected-attribute instruction.
	assert.Contains(t, Get(IDShortlistScore).Body, "never on protected attributes")
}

func TestRegistry_RequestStampsSourceAndBudget(t *testing.T) {
	p := Get(IDShortlistScore)
	req := p.Request("the user content")
	assert.Equal(t, p.Body, req.System)
	assert.Equal(t, "the user content", req.Prompt)
	assert.Equal(t, p.MaxTokens, req.MaxTokens)
	assert.Equal(t, "shortlist_score", req.Source.ID)
	assert.Equal(t, "v1", req.Source.Version)
}

func TestRegistry_GetUnknownPanics(t *testing.T) {
	assert.Panics(t, func() { Get("no_such_prompt") })
}

func TestRegistry_MustLoadFailsFast(t *testing.T) {
	assert.Panics(t, func() {
		mustLoad([]reg{{IDRoleSpec, "v1", "files/does_not_exist.txt", 1}})
	}, "missing file panics")
	assert.Panics(t, func() {
		mustLoad([]reg{
			{IDRoleSpec, "v1", "files/role_spec/v1.txt", 1},
			{IDRoleSpec, "v1", "files/role_spec/v1.txt", 1},
		})
	}, "duplicate id panics")
}
