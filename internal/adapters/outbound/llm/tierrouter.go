package llm

import (
	"context"

	"github.com/xcreativs/caliber/internal/app"
)

// TierRouter routes designated "cheap" operations to a cheaper model (CAL-159
// model-tier routing). It wraps an app.LLMClient and, when a cheap model is
// configured, sets LLMRequest.Model to it for calls whose prompt id is in the
// cheap set — mechanical operations (e.g. CV text extraction) that don't need
// the top-tier model. With no cheap model configured it is a passthrough, so it
// composes safely at the outermost layer of the LLM chain (so the audited
// recorder attributes spend to the model actually used).
type TierRouter struct {
	inner      app.LLMClient
	cheapModel string
	cheap      map[string]bool // prompt ids routed to the cheap model
}

// NewTierRouter wraps inner. cheapPromptIDs are the prompt Source ids routed to
// cheapModel; an empty cheapModel disables routing (passthrough).
func NewTierRouter(inner app.LLMClient, cheapModel string, cheapPromptIDs ...string) *TierRouter {
	set := make(map[string]bool, len(cheapPromptIDs))
	for _, id := range cheapPromptIDs {
		set[id] = true
	}
	return &TierRouter{inner: inner, cheapModel: cheapModel, cheap: set}
}

// Complete routes then delegates.
func (t *TierRouter) Complete(ctx context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	return t.inner.Complete(ctx, t.route(req))
}

// Stream routes then delegates.
func (t *TierRouter) Stream(ctx context.Context, req app.LLMRequest, yield app.LLMStreamYield) error {
	return t.inner.Stream(ctx, t.route(req), yield)
}

// Warm delegates unchanged (no request to route).
func (t *TierRouter) Warm(ctx context.Context) error { return t.inner.Warm(ctx) }

// route returns req with Model set to the cheap model when the call qualifies. A
// request that already pins a model is left untouched.
func (t *TierRouter) route(req app.LLMRequest) app.LLMRequest {
	if t.cheapModel != "" && req.Model == "" && t.cheap[req.Source.ID] {
		req.Model = t.cheapModel
	}
	return req
}

var _ app.LLMClient = (*TierRouter)(nil)
