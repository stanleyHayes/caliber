package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/app"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	rolesapp "github.com/xcreativs/caliber/internal/app/roles"
)

// TestAudited_ClassifiesRealSystemPrompts ties the operation classifier to the
// ACTUAL system-prompt constants the use-cases send. If a prompt is reworded so
// its classifier phrase changes, this test fails loudly instead of silently
// degrading every audit trace for that operation to "unknown".
func TestAudited_ClassifiesRealSystemPrompts(t *testing.T) {
	cases := []struct {
		system string
		want   string
	}{
		{interviewapp.QuestionSystemPrompt, "interview_question"},
		{interviewapp.ReportSystemPrompt, "interview_report"},
		{candidateagentapp.AgentSystemPrompt, "agent_assess"},
		{profilesapp.ExtractSystemPrompt, "cv_extract"},
		{matchingapp.ScoringSystemPrompt, "shortlist_score"},
		{rolesapp.SystemPrompt, "role_spec"},
	}
	for _, tc := range cases {
		rec := llm.NewMemoryRecorder(1)
		a := llm.NewAudited(stubLLM{}, rec, "dev", nil)
		_, _ = a.Complete(context.Background(), app.LLMRequest{System: tc.system})
		assert.Equalf(t, tc.want, rec.Snapshot()[0].Operation,
			"operationOf must classify the real %q system prompt", tc.want)
	}
}
