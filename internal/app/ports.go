package app

import (
	"context"
	"time"
)

// LLMClient is the application port for large-language-model access. All model
// interaction in the platform routes through this port (default impl: Claude).
type LLMClient interface {
	Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
	// Warm performs a lightweight, provider-specific pre-warm so the first real
	// interview question is served from an already-initialised session. It is a
	// no-op for providers that do not need warming (CAL-104).
	Warm(ctx context.Context) error
}

// PromptRef identifies the registry prompt a request was built from, so the
// audit trail records exactly which prompt id + version produced each call.
type PromptRef struct {
	ID      string
	Version string
}

// LLMRequest is a single completion request. Source is set when the request is
// built through the prompt registry (prompts.Prompt.Request). ExpectJSON marks
// structured-output calls so telemetry can track JSON failure rates (CAL-137).
type LLMRequest struct {
	System     string
	Prompt     string
	MaxTokens  int
	Source     PromptRef
	ExpectJSON bool
}

// LLMResponse is a completion result.
type LLMResponse struct {
	Text string
}

// Clock returns the current time; injectable for deterministic tests.
type Clock func() time.Time

// Embedder is the application port for producing vector embeddings (matching
// recall). The concrete provider (OpenAI today) is swappable.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

//go:generate mockgen -source=ports.go -destination=../mocks/llm.go -package=mocks
