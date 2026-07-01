package llm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/app"
)

func TestDevCompleteReturnsParseableSpec(t *testing.T) {
	resp, err := NewDev().Complete(context.Background(), app.LLMRequest{Prompt: "Need a data engineer\nmore detail"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(resp.Text), &doc); err != nil {
		t.Fatalf("dev output not JSON: %v", err)
	}
	if doc["title"] != "Need a data engineer" {
		t.Errorf("title = %v, want first line of prompt", doc["title"])
	}
	if _, ok := doc["rubric"]; !ok {
		t.Error("missing rubric")
	}
}

func TestDevCompleteEmptyPrompt(t *testing.T) {
	resp, _ := NewDev().Complete(context.Background(), app.LLMRequest{Prompt: "   "})
	var doc map[string]any
	_ = json.Unmarshal([]byte(resp.Text), &doc)
	if doc["title"] != "Software Engineer" {
		t.Errorf("empty prompt title = %v, want default", doc["title"])
	}
}

func TestDevExtractGroundsCompetenciesInCV(t *testing.T) {
	cv := "Senior engineer with Go, Postgres, and Kubernetes experience."
	doc := devExtract(cv)
	comps, ok := doc["competencies"].([]map[string]any)
	require.True(t, ok, "competencies is a slice")
	require.NotEmpty(t, comps)
	assert.Equal(t, "Core skills", comps[0][keyName])

	found := map[string]bool{}
	for _, c := range comps {
		name, ok := c[keyName].(string)
		require.True(t, ok, "competency name is a string")
		found[name] = true
	}
	assert.True(t, found["Go"], "Go is extracted from CV text")
	assert.True(t, found["Postgres"], "Postgres is extracted from CV text")
	assert.True(t, found["Kubernetes"], "Kubernetes is extracted from CV text")
}

func TestDevAgentTailorsSummaryFromVerifiedProfile(t *testing.T) {
	prompt := "ROLE: Mobile Engineer\nVERIFIED PROFILE COMPETENCIES:\n- React Native (level 5): shipped apps\n- Mobile (level 4): led team\n Assess fit."
	doc := devAgent(prompt)
	assert.InDelta(t, 0.8, doc["fit_score"], 1e-9)
	assert.Equal(t, true, doc["apply"])
	summary, ok := doc["tailored_summary"].(string)
	require.True(t, ok)
	assert.Contains(t, summary, "React Native")
	assert.Contains(t, summary, "Mobile")
}

func TestDevAgentSummaryWithoutCompetencies(t *testing.T) {
	doc := devAgent("ROLE: X\nNo verified profile block here.")
	assert.Equal(t, "Strong fit for this role based on verified experience.", doc["tailored_summary"])
}

func TestRoleTitleFromPromptBranches(t *testing.T) {
	// Fenced hiring need: title is the first non-empty line inside the fence.
	fenced := "Generate a role spec.\n[BEGIN UNTRUSTED HIRING_NEED]\n\nSenior Data Engineer\n[END UNTRUSTED HIRING_NEED]"
	assert.Equal(t, "Senior Data Engineer", roleTitleFromPrompt(fenced))

	// Empty fence returns empty string.
	emptyFence := "x\n[BEGIN UNTRUSTED HIRING_NEED]\n[END UNTRUSTED HIRING_NEED]"
	assert.Empty(t, roleTitleFromPrompt(emptyFence))

	// No fence: falls back to first line.
	assert.Equal(t, "Ad hoc title", roleTitleFromPrompt("Ad hoc title\nmore detail"))

	// Long first line is capped at 80 chars.
	long := strings.Repeat("a", 100)
	assert.Len(t, roleTitleFromPrompt(long), 80)
}
