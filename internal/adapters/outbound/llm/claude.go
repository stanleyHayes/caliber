package llm

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/xcreativs/caliber/internal/app"
)

// defaultMaxTokens caps a single completion when the caller does not specify one.
const defaultMaxTokens = 4096

// Claude is an app.LLMClient backed by the Anthropic Messages API. Default model
// is claude-opus-4-8 (overridable). All model access in the platform routes
// through the app.LLMClient port, so this is the only place the SDK is touched.
type Claude struct {
	client anthropic.Client
	model  string
}

type claudeConfig struct {
	apiKey  string
	baseURL string
	model   string
}

// ClaudeOption configures the Claude adapter.
type ClaudeOption func(*claudeConfig)

// WithAPIKey sets the Anthropic API key.
func WithAPIKey(k string) ClaudeOption { return func(c *claudeConfig) { c.apiKey = k } }

// WithBaseURL overrides the API base URL (used in tests).
func WithBaseURL(u string) ClaudeOption { return func(c *claudeConfig) { c.baseURL = u } }

// WithModel overrides the model id (ignored when empty).
func WithModel(m string) ClaudeOption {
	return func(c *claudeConfig) {
		if m != "" {
			c.model = m
		}
	}
}

// NewClaude builds a Claude adapter. Without options it reads ANTHROPIC_API_KEY
// from the environment and defaults to claude-opus-4-8.
func NewClaude(opts ...ClaudeOption) *Claude {
	cfg := claudeConfig{model: anthropic.ModelClaudeOpus4_8}
	for _, o := range opts {
		o(&cfg)
	}
	var reqOpts []option.RequestOption
	if cfg.apiKey != "" {
		reqOpts = append(reqOpts, option.WithAPIKey(cfg.apiKey))
	}
	if cfg.baseURL != "" {
		reqOpts = append(reqOpts, option.WithBaseURL(cfg.baseURL))
	}
	return &Claude{
		client: anthropic.NewClient(reqOpts...),
		model:  cfg.model,
	}
}

// Warm sends a tiny, throw-away completion to eagerly establish the provider
// connection before the interview starts (CAL-104). Errors are surfaced so the
// caller can fail fast instead of discovering a cold session on the first
// question.
func (c *Claude) Warm(ctx context.Context) error {
	_, err := c.Complete(ctx, app.LLMRequest{Prompt: "ping", MaxTokens: 1})
	return err
}

// Complete sends a single-turn message and returns the concatenated text blocks.
func (c *Claude) Complete(ctx context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	resp, err := c.client.Messages.New(ctx, c.params(req))
	if err != nil {
		return app.LLMResponse{}, err
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	return app.LLMResponse{Text: sb.String()}, nil
}

// Stream sends a single-turn message and yields text deltas as Anthropic emits
// them. It respects context cancellation through the SDK stream and stops early
// if yield returns an error, allowing inbound transports to apply backpressure.
func (c *Claude) Stream(ctx context.Context, req app.LLMRequest, yield app.LLMStreamYield) error {
	stream := c.client.Messages.NewStreaming(ctx, c.params(req))
	defer func() { _ = stream.Close() }()
	for stream.Next() {
		if ev, ok := stream.Current().AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			if delta, ok := ev.Delta.AsAny().(anthropic.TextDelta); ok && delta.Text != "" {
				if err := yield(app.LLMStreamEvent{Text: delta.Text}); err != nil {
					return err
				}
			}
		}
	}
	return stream.Err()
}

func (c *Claude) params(req app.LLMRequest) anthropic.MessageNewParams {
	maxTokens := int64(req.MaxTokens)
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	params := anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.Prompt)),
		},
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	return params
}

var _ app.LLMClient = (*Claude)(nil)
