package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// scriptedLLM returns queued responses/errors in order and records the prompts
// it was asked with.
type scriptedLLM struct {
	replies []reply
	prompts []string
	calls   int
}

type reply struct {
	text string
	err  error
}

func (s *scriptedLLM) Complete(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	s.prompts = append(s.prompts, req.Prompt)
	r := s.replies[min(s.calls, len(s.replies)-1)]
	s.calls++
	if r.err != nil {
		return app.LLMResponse{}, r.err
	}
	return app.LLMResponse{Text: r.text}, nil
}

func (s *scriptedLLM) Warm(_ context.Context) error { return nil }

type payload struct {
	Name string `json:"name"`
}

func TestDecodeJSON_SucceedsFirstTry(t *testing.T) {
	c := &scriptedLLM{replies: []reply{{text: `{"name":"Ama"}`}}}
	got, err := app.DecodeJSON[payload](context.Background(), c, app.LLMRequest{Prompt: "p"}, app.DefaultLLMAttempts, "test")
	require.NoError(t, err)
	assert.Equal(t, "Ama", got.Name)
	assert.Equal(t, 1, c.calls, "no re-ask when the first reply parses")
}

func TestDecodeJSON_ReAsksThenSucceeds(t *testing.T) {
	c := &scriptedLLM{replies: []reply{{text: "not json"}, {text: `{"name":"Kofi"}`}}}
	got, err := app.DecodeJSON[payload](context.Background(), c, app.LLMRequest{Prompt: "p"}, app.DefaultLLMAttempts, "test")
	require.NoError(t, err)
	assert.Equal(t, "Kofi", got.Name)
	require.Equal(t, 2, c.calls, "one re-ask after malformed output")
	assert.Contains(t, c.prompts[1], "could not be parsed as JSON", "the re-ask carries a corrective notice")
}

func TestDecodeJSON_ExhaustsAttempts(t *testing.T) {
	c := &scriptedLLM{replies: []reply{{text: "nope"}}}
	_, err := app.DecodeJSON[payload](context.Background(), c, app.LLMRequest{Prompt: "p"}, 3, "test")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
	assert.Equal(t, 3, c.calls, "tries the full attempt budget before giving up")
}

func TestDecodeJSON_TransportErrorIsInternalAndNotRetried(t *testing.T) {
	c := &scriptedLLM{replies: []reply{{err: errors.New("boom")}}}
	_, err := app.DecodeJSON[payload](context.Background(), c, app.LLMRequest{Prompt: "p"}, app.DefaultLLMAttempts, "test")
	assert.Equal(t, kernel.KindInternal, kernel.KindOf(err))
	assert.Equal(t, 1, c.calls, "a transport failure is not re-asked")
}

func TestDecodeJSON_AttemptsFlooredToOne(t *testing.T) {
	c := &scriptedLLM{replies: []reply{{text: "bad"}}}
	_, err := app.DecodeJSON[payload](context.Background(), c, app.LLMRequest{Prompt: "p"}, 0, "test")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
	assert.Equal(t, 1, c.calls, "attempts < 1 is treated as a single attempt")
}

func TestDecodeJSON_ErrorLabelled(t *testing.T) {
	c := &scriptedLLM{replies: []reply{{text: "bad"}}}
	_, err := app.DecodeJSON[payload](context.Background(), c, app.LLMRequest{Prompt: "p"}, 1, "interview: report")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "interview: report:"), "error is prefixed with the label")
}
