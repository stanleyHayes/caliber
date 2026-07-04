package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/app"
)

// recordingLLM captures the request it was called with.
type recordingLLM struct{ got app.LLMRequest }

func (r *recordingLLM) Complete(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	r.got = req
	return app.LLMResponse{Text: "ok"}, nil
}

func (r *recordingLLM) Stream(_ context.Context, req app.LLMRequest, _ app.LLMStreamYield) error {
	r.got = req
	return nil
}

func (r *recordingLLM) Warm(context.Context) error { return nil }

func req(id string) app.LLMRequest {
	return app.LLMRequest{Prompt: "x", Source: app.PromptRef{ID: id, Version: "v1"}}
}

func TestTierRouter_RoutesCheapOpToCheapModel(t *testing.T) {
	inner := &recordingLLM{}
	router := llm.NewTierRouter(inner, "claude-haiku-4-5", "cv_extract")

	_, err := router.Complete(context.Background(), req("cv_extract"))
	require.NoError(t, err)
	assert.Equal(t, "claude-haiku-4-5", inner.got.Model, "a cheap op is routed to the cheap model")

	// A non-cheap op keeps the default (empty -> provider default downstream).
	_, err = router.Complete(context.Background(), req("interview_question"))
	require.NoError(t, err)
	assert.Empty(t, inner.got.Model, "the interview op stays on the default model")
}

func TestTierRouter_PassthroughWhenNoCheapModel(t *testing.T) {
	inner := &recordingLLM{}
	router := llm.NewTierRouter(inner, "", "cv_extract") // routing disabled

	_, err := router.Complete(context.Background(), req("cv_extract"))
	require.NoError(t, err)
	assert.Empty(t, inner.got.Model, "no routing when no cheap model is configured")
}

func TestTierRouter_DoesNotOverrideAnExplicitModel(t *testing.T) {
	inner := &recordingLLM{}
	router := llm.NewTierRouter(inner, "claude-haiku-4-5", "cv_extract")
	r := req("cv_extract")
	r.Model = "claude-opus-4-8" // caller pinned a model
	_, err := router.Complete(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-8", inner.got.Model, "an explicit model is respected")
}

func TestTierRouter_StreamRoutesToo(t *testing.T) {
	inner := &recordingLLM{}
	router := llm.NewTierRouter(inner, "claude-haiku-4-5", "cv_extract")
	err := router.Stream(context.Background(), req("cv_extract"), func(app.LLMStreamEvent) error { return nil })
	require.NoError(t, err)
	assert.Equal(t, "claude-haiku-4-5", inner.got.Model)
}
