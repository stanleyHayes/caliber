package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyOptsDefaults(t *testing.T) {
	o := ApplyOpts()
	assert.Equal(t, -1, o.MaxRetry)
	assert.Equal(t, QueueDefault, o.Queue)
	assert.Zero(t, o.ProcessIn)
	assert.Zero(t, o.UniqueTTL)
}

func TestApplyOptsOverride(t *testing.T) {
	at := time.Now().Add(time.Hour)
	o := ApplyOpts(
		ProcessIn(5*time.Minute),
		ProcessAt(at),
		Unique(10*time.Minute),
		MaxRetry(7),
		Queue(QueueCritical),
	)
	assert.Equal(t, 5*time.Minute, o.ProcessIn)
	assert.WithinDuration(t, at, o.ProcessAt, 0)
	assert.Equal(t, 10*time.Minute, o.UniqueTTL)
	assert.Equal(t, 7, o.MaxRetry)
	assert.Equal(t, QueueCritical, o.Queue)
}

func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, TypeCandidateAgentRun, TaskType("candidate_agent:run"))
	assert.Equal(t, TypeInterviewScoring, TaskType("interview:score"))
	assert.Equal(t, TypeBatchRematch, TaskType("matching:rematch"))
}

func TestDefaultRetryPolicyPerTaskType(t *testing.T) {
	cases := []struct {
		typ            TaskType
		wantMaxRetry   int
		wantInitial    time.Duration
		wantMaxDelay   time.Duration
		wantJitter     float64
	}{
		{TypeCandidateAgentRun, 3, 5 * time.Second, 5 * time.Minute, 0.2},
		{TypeInterviewScoring, 3, 5 * time.Second, 5 * time.Minute, 0.2},
		{TypeBatchRematch, 2, 10 * time.Second, 2 * time.Minute, 0.2},
		{TaskType("unknown"), 3, 10 * time.Second, 1 * time.Minute, 0.2},
	}
	for _, tc := range cases {
		policy := DefaultRetryPolicy(tc.typ)
		assert.Equal(t, tc.wantMaxRetry, policy.MaxRetry, "type=%s", tc.typ)
		assert.Equal(t, tc.wantInitial, policy.InitialDelay, "type=%s", tc.typ)
		assert.Equal(t, tc.wantMaxDelay, policy.MaxDelay, "type=%s", tc.typ)
		assert.InDelta(t, tc.wantJitter, policy.Jitter, 0.001, "type=%s", tc.typ)
	}
}

func TestComputeBackoffExponentialGrowthAndCap(t *testing.T) {
	policy := RetryPolicy{
		MaxRetry:     3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Jitter:       0,
	}

	assert.Equal(t, 100*time.Millisecond, ComputeBackoff(policy, 0))
	assert.Equal(t, 200*time.Millisecond, ComputeBackoff(policy, 1))
	assert.Equal(t, 400*time.Millisecond, ComputeBackoff(policy, 2))
	assert.Equal(t, 500*time.Millisecond, ComputeBackoff(policy, 3), "delay must be capped at MaxDelay")
	assert.Equal(t, 500*time.Millisecond, ComputeBackoff(policy, 10), "delay stays capped")
}

func TestComputeBackoffJitter(t *testing.T) {
	policy := RetryPolicy{
		MaxRetry:     3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Jitter:       0.5,
	}

	for i := 0; i < 20; i++ {
		delay := ComputeBackoff(policy, 1)
		assert.GreaterOrEqual(t, delay, 150*time.Millisecond, "jittered delay must be >= base - spread/2")
		assert.LessOrEqual(t, delay, 250*time.Millisecond, "jittered delay must be <= base + spread/2")
	}
}

func TestComputeBackoffSanitizesInvalidPolicy(t *testing.T) {
	policy := RetryPolicy{
		MaxRetry:     1,
		InitialDelay: 0,
		MaxDelay:     -1,
		Jitter:       0,
	}
	// Invalid policy is sanitized to at least 1s initial and equal max.
	delay := ComputeBackoff(policy, 0)
	assert.GreaterOrEqual(t, delay, time.Duration(0))
	assert.LessOrEqual(t, delay, time.Second)
}
