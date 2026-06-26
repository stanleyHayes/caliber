package llm_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/app"
)

func TestDevShapesInterviewResponses(t *testing.T) {
	d := llm.NewDev()
	ctx := context.Background()

	q, err := d.Complete(ctx, app.LLMRequest{
		System: "You are an adaptive technical screening interviewer.",
		Prompt: "ROLE: Backend\nRUBRIC:\n- Go\n- SQL\nTRANSCRIPT: (none yet)\nAsk the next question.",
	})
	require.NoError(t, err)
	var question struct {
		Question      string `json:"question"`
		CompetencyTag string `json:"competency_tag"`
	}
	require.NoError(t, json.Unmarshal([]byte(q.Text), &question))
	assert.NotEmpty(t, question.Question)
	assert.Equal(t, "Go", question.CompetencyTag, "targets the first rubric competency on turn 0")

	r, err := d.Complete(ctx, app.LLMRequest{
		System: "You score a screening interview against the role rubric.",
		Prompt: "ROLE: Backend\nRUBRIC:\n- Go\nTRANSCRIPT:\nQ1 (Go): tell me\nA: I built a payments service\nScore the interview now.",
	})
	require.NoError(t, err)
	var report struct {
		Verdict string `json:"verdict"`
		Scores  []struct {
			Competency string  `json:"competency"`
			Score      float64 `json:"score"`
			Evidence   string  `json:"evidence"`
		} `json:"scores"`
	}
	require.NoError(t, json.Unmarshal([]byte(r.Text), &report))
	assert.Equal(t, "advance", report.Verdict)
	require.NotEmpty(t, report.Scores)
	assert.Equal(t, "I built a payments service", report.Scores[0].Evidence, "evidence quotes the candidate (no fabrication)")
}
