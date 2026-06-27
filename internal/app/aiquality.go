package app

import (
	"math"
	"slices"
	"time"
)

// AIQualityStats summarizes a window of AICallRecords for AI-quality monitoring
// (CAL-137): call volume, failure rate, latency percentiles, and a token-proxy
// cost signal, broken down per logical operation. It is computed from the
// redacted call traces (sizes/latency/outcome only), so it carries no PII and is
// safe to surface in dashboards or logs.
type AIQualityStats struct {
	TotalCalls  int
	FailedCalls int
	FailureRate float64 // FailedCalls / TotalCalls, 0 when there are no calls
	P50Latency  time.Duration
	P95Latency  time.Duration
	InputChars  int // sum of prompt sizes (proxy for input tokens / cost)
	OutputChars int // sum of response sizes (proxy for output tokens / cost)
	ByOperation map[string]OperationStats
}

// OperationStats is the per-operation slice of AIQualityStats.
type OperationStats struct {
	Calls       int
	Failed      int
	FailureRate float64
	P95Latency  time.Duration
}

// SummarizeAIQuality computes AIQualityStats over the given call records. It is
// pure and order-independent; an empty slice yields a zero-value summary with an
// empty (non-nil) ByOperation map.
func SummarizeAIQuality(records []AICallRecord) AIQualityStats {
	stats := AIQualityStats{ByOperation: make(map[string]OperationStats)}
	if len(records) == 0 {
		return stats
	}

	latencies := make([]time.Duration, 0, len(records))
	perOp := make(map[string][]time.Duration)
	perOpFailed := make(map[string]int)
	for _, rec := range records {
		stats.TotalCalls++
		stats.InputChars += rec.PromptChars
		stats.OutputChars += rec.ResponseChars
		latencies = append(latencies, rec.Latency)
		perOp[rec.Operation] = append(perOp[rec.Operation], rec.Latency)
		if rec.Failed {
			stats.FailedCalls++
			perOpFailed[rec.Operation]++
		}
	}

	stats.FailureRate = ratio(stats.FailedCalls, stats.TotalCalls)
	stats.P50Latency = percentile(latencies, 0.50)
	stats.P95Latency = percentile(latencies, 0.95)
	for op, lat := range perOp {
		stats.ByOperation[op] = OperationStats{
			Calls:       len(lat),
			Failed:      perOpFailed[op],
			FailureRate: ratio(perOpFailed[op], len(lat)),
			P95Latency:  percentile(lat, 0.95),
		}
	}
	return stats
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
