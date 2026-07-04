package app

import (
	"sort"
	"strings"
	"sync"
)

// charsPerToken approximates the tokenizer. The platform records call sizes in
// characters (a PII-safe proxy — never content), and ~4 characters ≈ 1 token for
// English prose. Cost is therefore an estimate until a real tokenizer is wired,
// which is sufficient for budget alerting and fail-fast capping (CAL-159).
const charsPerToken = 4.0

// ModelPricing is the USD price per one million input/output tokens for a model.
type ModelPricing struct {
	InputPerMillionUSD  float64
	OutputPerMillionUSD float64
}

// pricingFor returns the price for a model id, matched by family prefix. The dev
// provider and unknown-but-empty models are free so local/offline runs never
// accrue cost; a genuinely unknown model is priced like Opus so a budget trips
// early rather than silently under-counting. Prices are Anthropic list prices.
func pricingFor(model string) ModelPricing {
	switch {
	case model == "" || model == "dev":
		return ModelPricing{}
	// OpenAI embeddings are input-only (no output tokens). List prices per 1M tokens.
	case strings.HasPrefix(model, "text-embedding-3-large"):
		return ModelPricing{InputPerMillionUSD: 0.13}
	case strings.HasPrefix(model, "text-embedding"):
		return ModelPricing{InputPerMillionUSD: 0.02} // 3-small / ada-002 tier
	case strings.HasPrefix(model, "claude-fable"), strings.HasPrefix(model, "claude-mythos"):
		return ModelPricing{InputPerMillionUSD: 10, OutputPerMillionUSD: 50}
	case strings.HasPrefix(model, "claude-opus"):
		return ModelPricing{InputPerMillionUSD: 5, OutputPerMillionUSD: 25}
	case strings.HasPrefix(model, "claude-sonnet"):
		return ModelPricing{InputPerMillionUSD: 3, OutputPerMillionUSD: 15}
	case strings.HasPrefix(model, "claude-haiku"):
		return ModelPricing{InputPerMillionUSD: 1, OutputPerMillionUSD: 5}
	default:
		return ModelPricing{InputPerMillionUSD: 5, OutputPerMillionUSD: 25}
	}
}

// EstimateCallCostUSD estimates the USD cost of one recorded model call from its
// character sizes (token counts proxied by chars/4). A dev/free model is $0.
func EstimateCallCostUSD(rec AICallRecord) float64 {
	p := pricingFor(rec.Model)
	inputTokens := float64(rec.PromptChars) / charsPerToken
	outputTokens := float64(rec.ResponseChars) / charsPerToken
	return inputTokens/1e6*p.InputPerMillionUSD + outputTokens/1e6*p.OutputPerMillionUSD
}

// CostAlert describes a budget event passed to a CostTracker's alert callback.
type CostAlert struct {
	SpentUSD  float64
	BudgetUSD float64
	Fraction  float64 // SpentUSD / BudgetUSD at the moment the threshold fired
	Exceeded  bool    // spend has reached or passed the full budget
}

// CostTracker accumulates estimated LLM spend and raises an alert each time a
// configured budget threshold is first crossed (CAL-159). It implements
// AICallRecorder, so wiring it as one of the audited recorders makes it observe
// every model call, and WithinBudget lets a guard fail calls fast once the
// budget is spent. Safe for concurrent use.
type CostTracker struct {
	mu         sync.Mutex
	budgetUSD  float64
	spentUSD   float64
	thresholds []float64 // ascending fractions in (0,1]
	fired      map[float64]bool
	alert      func(CostAlert)
}

// NewCostTracker builds a tracker with the given budget (USD; <=0 means unlimited
// — observe-only, never caps) and alert callback. Thresholds are budget fractions
// at which to alert; when none are given it defaults to 50%, 80%, and 100%.
func NewCostTracker(budgetUSD float64, alert func(CostAlert), thresholds ...float64) *CostTracker {
	if len(thresholds) == 0 {
		thresholds = []float64{0.5, 0.8, 1.0}
	}
	sorted := append([]float64(nil), thresholds...)
	sort.Float64s(sorted)
	return &CostTracker{
		budgetUSD:  budgetUSD,
		thresholds: sorted,
		fired:      make(map[float64]bool, len(sorted)),
		alert:      alert,
	}
}

// Record adds the estimated cost of one call and fires any newly-crossed budget
// threshold alerts. Each threshold alerts at most once. A free ($0) call is a
// no-op. Alerts are dispatched outside the lock so a slow callback cannot stall
// concurrent recorders.
func (c *CostTracker) Record(rec AICallRecord) {
	cost := EstimateCallCostUSD(rec)
	if cost <= 0 {
		return
	}
	c.mu.Lock()
	c.spentUSD += cost
	spent, budget := c.spentUSD, c.budgetUSD
	var alerts []CostAlert
	if budget > 0 {
		fraction := spent / budget
		for _, t := range c.thresholds {
			if fraction >= t && !c.fired[t] {
				c.fired[t] = true
				alerts = append(alerts, CostAlert{
					SpentUSD:  spent,
					BudgetUSD: budget,
					Fraction:  fraction,
					Exceeded:  fraction >= 1,
				})
			}
		}
	}
	c.mu.Unlock()
	if c.alert != nil {
		for _, a := range alerts {
			c.alert(a)
		}
	}
}

// SpentUSD returns the cumulative estimated spend so far.
func (c *CostTracker) SpentUSD() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.spentUSD
}

// WithinBudget reports whether more spend is permitted. A non-positive budget is
// treated as unlimited, so the tracker can run in observe-only mode.
func (c *CostTracker) WithinBudget() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.budgetUSD <= 0 {
		return true
	}
	return c.spentUSD < c.budgetUSD
}
