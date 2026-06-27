package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/app"
)

func rec(op string, latencyMs int, failed bool, in, out int) app.AICallRecord {
	return app.AICallRecord{
		Operation:     op,
		Latency:       time.Duration(latencyMs) * time.Millisecond,
		Failed:        failed,
		PromptChars:   in,
		ResponseChars: out,
	}
}

func TestSummarizeAIQuality_Empty(t *testing.T) {
	stats := app.SummarizeAIQuality(nil)
	assert.Zero(t, stats.TotalCalls)
	assert.Zero(t, stats.FailureRate)
	assert.NotNil(t, stats.ByOperation, "ByOperation is an empty map, not nil")
	assert.Empty(t, stats.ByOperation)
}

func TestSummarizeAIQuality_AggregatesAndRates(t *testing.T) {
	records := []app.AICallRecord{
		rec("score", 100, false, 10, 20),
		rec("score", 300, true, 12, 0),
		rec("interview", 200, false, 8, 16),
		rec("interview", 400, false, 9, 18),
	}
	stats := app.SummarizeAIQuality(records)

	assert.Equal(t, 4, stats.TotalCalls)
	assert.Equal(t, 1, stats.FailedCalls)
	assert.InDelta(t, 0.25, stats.FailureRate, 1e-9)
	assert.Equal(t, 39, stats.InputChars)
	assert.Equal(t, 54, stats.OutputChars)

	// score: 1 of 2 failed.
	score := stats.ByOperation["score"]
	assert.Equal(t, 2, score.Calls)
	assert.Equal(t, 1, score.Failed)
	assert.InDelta(t, 0.5, score.FailureRate, 1e-9)

	// interview: 0 of 2 failed.
	interview := stats.ByOperation["interview"]
	assert.Equal(t, 2, interview.Calls)
	assert.Zero(t, interview.Failed)
}

func TestSummarizeAIQuality_LatencyPercentiles(t *testing.T) {
	// Latencies 100..1000ms; nearest-rank p50 -> 500ms (rank 4 of 10), p95 -> 1000ms.
	var records []app.AICallRecord
	for i := 1; i <= 10; i++ {
		records = append(records, rec("score", i*100, false, 1, 1))
	}
	stats := app.SummarizeAIQuality(records)
	assert.Equal(t, 500*time.Millisecond, stats.P50Latency)
	assert.Equal(t, 1000*time.Millisecond, stats.P95Latency)
}

func TestSummarizeAIQuality_SingleRecordPercentile(t *testing.T) {
	stats := app.SummarizeAIQuality([]app.AICallRecord{rec("x", 250, false, 1, 1)})
	assert.Equal(t, 250*time.Millisecond, stats.P50Latency)
	assert.Equal(t, 250*time.Millisecond, stats.P95Latency)
}
