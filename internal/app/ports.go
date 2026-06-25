package app

import (
	"context"
	"time"
)

// LLMClient is the application port for large-language-model access. All model
// interaction in the platform routes through this port (default impl: Claude).
type LLMClient interface {
	Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

// LLMRequest is a single completion request.
type LLMRequest struct {
	System    string
	Prompt    string
	MaxTokens int
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
