package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
)

// sequencedLLM returns its replies in order (sticking on the last), so a first
// malformed reply forces app.DecodeJSON to re-ask.
type sequencedLLM struct {
	replies []string
	i       int
}

func (s *sequencedLLM) Complete(_ context.Context, _ app.LLMRequest) (app.LLMResponse, error) {
	r := s.replies[min(s.i, len(s.replies)-1)]
	s.i++
	return app.LLMResponse{Text: r}, nil
}

func (s *sequencedLLM) Warm(_ context.Context) error { return nil }

// TestAudited_PromptSourceSurvivesReAskLoop is the CAL-032 acceptance guard: the
// prompt id+version must be recorded on EVERY model call, including the retries
// app.DecodeJSON makes after a malformed reply. It wraps a stub whose first reply
// is unparseable in the real Audited decorator, drives it through DecodeJSON with
// a registry-built request, and asserts both attempts recorded the same non-empty
// id+version. A future regression that rebuilt the request inside the re-ask loop
// (dropping Source) would record "unknown" on the second attempt and fail here.
func TestAudited_PromptSourceSurvivesReAskLoop(t *testing.T) {
	inner := &sequencedLLM{replies: []string{
		"not json", // attempt 1: forces a re-ask
		`{"verdict":"advance","confidence":"low","scores":[]}`, // attempt 2: parses
	}}
	rec := llm.NewMemoryRecorder(4)
	audited := llm.NewAudited(inner, rec, "dev", nil)

	type report struct {
		Verdict string `json:"verdict"`
	}
	_, err := app.DecodeJSON[report](context.Background(), audited,
		prompts.Get(prompts.IDInterviewReport).Request("score the interview now"),
		app.DefaultLLMAttempts, "interview: report")
	require.NoError(t, err)

	snap := rec.Snapshot()
	require.Len(t, snap, 2, "one audit record per attempt across the re-ask loop")
	for i, rcd := range snap {
		assert.Equalf(t, "interview_report", rcd.PromptID, "attempt %d records the prompt id", i+1)
		assert.NotEmptyf(t, rcd.PromptVersion, "attempt %d records a version", i+1)
	}
	assert.Equal(t, snap[0].PromptID, snap[1].PromptID, "Source id is stable across the re-ask")
	assert.Equal(t, snap[0].PromptVersion, snap[1].PromptVersion, "Source version is stable across the re-ask")
}
