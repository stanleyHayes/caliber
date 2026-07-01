package app

import (
	"encoding/json"
	"maps"
	"math"
	"slices"
	"strings"
	"time"
)

// AIQualityStats summarizes a window of AICallRecords for AI-quality monitoring
// (CAL-137): call volume, failure rate, latency percentiles, a token-proxy cost
// signal, structured-output (JSON) failure rate, refusal rate, and guardrail
// trip counts, broken down per logical operation. It is computed from the
// redacted call traces (sizes/latency/outcome only), so it carries no PII and is
// safe to surface in dashboards or logs.
type AIQualityStats struct {
	TotalCalls               int
	FailedCalls              int
	FailureRate              float64        // FailedCalls / TotalCalls
	JSONFailures             int            // calls where JSON parsing failed
	JSONFailureRate          float64        // JSONFailures / TotalCalls
	Refusals                 int            // calls detected as refusals
	RefusalRate              float64        // Refusals / TotalCalls
	GuardrailTrips           int            // total guardrail trips across all calls
	GuardrailTripsByCategory map[string]int // trips aggregated by category label
	P50Latency               time.Duration
	P95Latency               time.Duration
	InputChars               int // sum of prompt sizes (proxy for input tokens / cost)
	OutputChars              int // sum of response sizes (proxy for output tokens / cost)
	ByOperation              map[string]OperationStats
}

// OperationStats is the per-operation slice of AIQualityStats.
type OperationStats struct {
	Calls                    int
	Failed                   int
	FailureRate              float64
	JSONFailures             int
	JSONFailureRate          float64
	Refusals                 int
	RefusalRate              float64
	GuardrailTrips           int
	GuardrailTripsByCategory map[string]int
	P95Latency               time.Duration
}

// SummarizeAIQuality computes AIQualityStats over the given call records. It is
// pure and order-independent; an empty slice yields a zero-value summary with an
// empty (non-nil) ByOperation map and empty guardrail category maps.
func SummarizeAIQuality(records []AICallRecord) AIQualityStats {
	stats := AIQualityStats{
		ByOperation:              make(map[string]OperationStats),
		GuardrailTripsByCategory: make(map[string]int),
	}
	if len(records) == 0 {
		return stats
	}

	latencies := make([]time.Duration, 0, len(records))
	perOp := make(map[string][]time.Duration)
	perOpFailed := make(map[string]int)
	perOpJSONFailures := make(map[string]int)
	perOpRefusals := make(map[string]int)
	perOpGuardrailTrips := make(map[string]int)
	perOpGuardrailCategories := make(map[string]map[string]int)

	for _, rec := range records {
		stats, latencies = accumulateRecord(
			stats, rec, latencies, perOp,
			perOpFailed, perOpJSONFailures, perOpRefusals,
			perOpGuardrailTrips, perOpGuardrailCategories,
		)
	}

	stats.FailureRate = ratio(stats.FailedCalls, stats.TotalCalls)
	stats.JSONFailureRate = ratio(stats.JSONFailures, stats.TotalCalls)
	stats.RefusalRate = ratio(stats.Refusals, stats.TotalCalls)
	stats.P50Latency = percentile(latencies, 0.50)
	stats.P95Latency = percentile(latencies, 0.95)
	for op, lat := range perOp {
		stats.ByOperation[op] = buildOperationStats(
			op, lat, perOpFailed, perOpJSONFailures,
			perOpRefusals, perOpGuardrailTrips, perOpGuardrailCategories,
		)
	}
	return stats
}

func accumulateRecord(
	stats AIQualityStats,
	rec AICallRecord,
	latencies []time.Duration,
	perOp map[string][]time.Duration,
	perOpFailed, perOpJSONFailures, perOpRefusals, perOpGuardrailTrips map[string]int,
	perOpGuardrailCategories map[string]map[string]int,
) (AIQualityStats, []time.Duration) {
	stats.TotalCalls++
	stats.InputChars += rec.PromptChars
	stats.OutputChars += rec.ResponseChars
	latencies = append(latencies, rec.Latency)
	perOp[rec.Operation] = append(perOp[rec.Operation], rec.Latency)
	if rec.Failed {
		stats.FailedCalls++
		perOpFailed[rec.Operation]++
	}
	if rec.JSONFailure {
		stats.JSONFailures++
		perOpJSONFailures[rec.Operation]++
	}
	if rec.Refusal {
		stats.Refusals++
		perOpRefusals[rec.Operation]++
	}
	if len(rec.GuardrailTrips) > 0 {
		stats.GuardrailTrips += len(rec.GuardrailTrips)
		perOpGuardrailTrips[rec.Operation] += len(rec.GuardrailTrips)
		if _, ok := perOpGuardrailCategories[rec.Operation]; !ok {
			perOpGuardrailCategories[rec.Operation] = make(map[string]int)
		}
		for _, cat := range rec.GuardrailTrips {
			stats.GuardrailTripsByCategory[cat]++
			perOpGuardrailCategories[rec.Operation][cat]++
		}
	}
	return stats, latencies
}

func buildOperationStats(
	op string,
	lat []time.Duration,
	perOpFailed, perOpJSONFailures, perOpRefusals, perOpGuardrailTrips map[string]int,
	perOpGuardrailCategories map[string]map[string]int,
) OperationStats {
	calls := len(lat)
	opStats := OperationStats{
		Calls:           calls,
		Failed:          perOpFailed[op],
		FailureRate:     ratio(perOpFailed[op], calls),
		JSONFailures:    perOpJSONFailures[op],
		JSONFailureRate: ratio(perOpJSONFailures[op], calls),
		Refusals:        perOpRefusals[op],
		RefusalRate:     ratio(perOpRefusals[op], calls),
		GuardrailTrips:  perOpGuardrailTrips[op],
		P95Latency:      percentile(lat, 0.95),
	}
	return finalizeOperationStats(opStats, perOpGuardrailCategories[op])
}

func finalizeOperationStats(opStats OperationStats, cats map[string]int) OperationStats {
	opStats.GuardrailTripsByCategory = make(map[string]int)
	if len(cats) > 0 {
		maps.Copy(opStats.GuardrailTripsByCategory, cats)
	}
	return opStats
}

func ratio(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

// percentile returns the nearest-rank pth percentile (0..1) of the durations.
func percentile(durations []time.Duration, p float64) time.Duration {
	n := len(durations)
	if n == 0 {
		return 0
	}
	sorted := make([]time.Duration, n)
	copy(sorted, durations)
	slices.Sort(sorted)
	rank := int(math.Ceil(p*float64(n))) - 1
	rank = min(max(rank, 0), n-1)
	return sorted[rank]
}

// LooksLikeRefusal returns true when text contains common refusal language.
// It is intentionally conservative (pattern-based, not semantic) and is used for
// telemetry only.
func LooksLikeRefusal(text string) bool {
	lower := strings.ToLower(text)
	phrases := []string{
		"i'm sorry", "i am sorry", "i cannot", "i can't", "i am not able",
		"i'm not able", "i will not", "i won't", "i cannot assist", "i can't assist",
		"i am unable to", "i'm unable to", "i refuse", "i do not", "i don't",
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// IsValidJSON reports whether s is well-formed JSON.
func IsValidJSON(s string) bool {
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}
