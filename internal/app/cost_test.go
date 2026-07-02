package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/app"
)

func TestEstimateCallCostUSD(t *testing.T) {
	// 4000 input chars ≈ 1000 tokens, 400 output chars ≈ 100 tokens.
	// Opus: 1000/1e6*$5 + 100/1e6*$25 = 0.005 + 0.0025 = 0.0075.
	opus := app.EstimateCallCostUSD(app.AICallRecord{Model: "claude-opus-4-8", PromptChars: 4000, ResponseChars: 400})
	assert.InDelta(t, 0.0075, opus, 1e-9)

	// Haiku is cheaper than Opus for the same sizes.
	haiku := app.EstimateCallCostUSD(app.AICallRecord{Model: "claude-haiku-4-5", PromptChars: 4000, ResponseChars: 400})
	assert.Less(t, haiku, opus)

	// The dev provider and empty model are free.
	assert.Zero(t, app.EstimateCallCostUSD(app.AICallRecord{Model: "dev", PromptChars: 8000, ResponseChars: 8000}))
	assert.Zero(t, app.EstimateCallCostUSD(app.AICallRecord{Model: "", PromptChars: 8000}))

	// An unknown model is priced conservatively (like Opus) rather than free.
	unknown := app.EstimateCallCostUSD(app.AICallRecord{Model: "mystery-1", PromptChars: 4000, ResponseChars: 400})
	assert.InDelta(t, opus, unknown, 1e-9)
}

func TestCostTracker_AccumulatesAndGates(t *testing.T) {
	var alerts []app.CostAlert
	// Budget $0.01; thresholds 50% and 100%.
	tracker := app.NewCostTracker(0.01, func(a app.CostAlert) { alerts = append(alerts, a) }, 0.5, 1.0)

	assert.True(t, tracker.WithinBudget(), "starts within budget")
	assert.Zero(t, tracker.SpentUSD())

	// One Opus call at $0.0075 crosses 50% but not 100%.
	call := app.AICallRecord{Model: "claude-opus-4-8", PromptChars: 4000, ResponseChars: 400}
	tracker.Record(call)
	assert.InDelta(t, 0.0075, tracker.SpentUSD(), 1e-9)
	require.Len(t, alerts, 1)
	assert.InDelta(t, 0.5, alerts[0].Fraction, 0.3)
	assert.False(t, alerts[0].Exceeded)
	assert.True(t, tracker.WithinBudget(), "still under $0.01")

	// A second identical call takes spend to $0.015 > budget: 100% alert fires and
	// the gate closes.
	tracker.Record(call)
	require.Len(t, alerts, 2)
	assert.True(t, alerts[1].Exceeded)
	assert.False(t, tracker.WithinBudget(), "over budget → gate closed")

	// A third call adds cost but fires no further alerts (each threshold once).
	tracker.Record(call)
	assert.Len(t, alerts, 2)
}

func TestCostTracker_ZeroBudgetIsUnlimited(t *testing.T) {
	tracker := app.NewCostTracker(0, nil) // observe-only
	tracker.Record(app.AICallRecord{Model: "claude-opus-4-8", PromptChars: 40000, ResponseChars: 40000})
	assert.Positive(t, tracker.SpentUSD(), "spend is still tracked")
	assert.True(t, tracker.WithinBudget(), "a zero budget never caps")
}

func TestCostTracker_FreeCallsDoNotAlert(t *testing.T) {
	var alerts []app.CostAlert
	tracker := app.NewCostTracker(0.01, func(a app.CostAlert) { alerts = append(alerts, a) })
	tracker.Record(app.AICallRecord{Model: "dev", PromptChars: 999999})
	assert.Zero(t, tracker.SpentUSD())
	assert.Empty(t, alerts)
}
