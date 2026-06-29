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
	o := ApplyOpts(
		ProcessIn(5*time.Minute),
		Unique(10*time.Minute),
		MaxRetry(7),
		Queue(QueueCritical),
	)
	assert.Equal(t, 5*time.Minute, o.ProcessIn)
	assert.Equal(t, 10*time.Minute, o.UniqueTTL)
	assert.Equal(t, 7, o.MaxRetry)
	assert.Equal(t, QueueCritical, o.Queue)
}

func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, TypeCandidateAgentRun, TaskType("candidate_agent:run"))
	assert.Equal(t, TypeInterviewScoring, TaskType("interview:score"))
	assert.Equal(t, TypeBatchRematch, TaskType("matching:rematch"))
}
