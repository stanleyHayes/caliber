package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestRedisOptParsesURL(t *testing.T) {
	redis := miniredis.RunT(t)
	opt, err := queue.RedisOpt("redis://" + redis.Addr() + "/0")
	require.NoError(t, err)
	assert.NotNil(t, opt)
}

func TestPrioritiesHasDefaultQueue(t *testing.T) {
	p := queue.Priorities()
	assert.Positive(t, p["default"])
}

func TestNoopDispatcherReturnsEmptyTaskID(t *testing.T) {
	d := queue.NewNoop()
	id, err := d.DispatchCandidateAgentRun(context.Background(), kernel.NewID())
	require.NoError(t, err)
	assert.Empty(t, id)
	require.NoError(t, d.Close())
	assert.True(t, queue.IsNoop(d))
}

func TestDispatcherEnqueuesCandidateAgentRun(t *testing.T) {
	redis := miniredis.RunT(t)
	d, err := queue.NewDispatcher("redis://" + redis.Addr() + "/0")
	require.NoError(t, err)
	defer func() { require.NoError(t, d.Close()) }()

	taskID, err := d.DispatchCandidateAgentRun(context.Background(), kernel.NewID())
	require.NoError(t, err)
	assert.NotEmpty(t, taskID)
}

func TestDispatcherEnqueuesWithOptions(t *testing.T) {
	redis := miniredis.RunT(t)
	d, err := queue.NewDispatcher("redis://" + redis.Addr() + "/0")
	require.NoError(t, err)
	defer func() { require.NoError(t, d.Close()) }()

	_, err = d.DispatchInterviewScoring(context.Background(), kernel.NewID(),
		appqueue.ProcessIn(time.Second),
		appqueue.MaxRetry(5),
		appqueue.Queue(appqueue.QueueCritical),
	)
	require.NoError(t, err)

	// Inspect the queue to confirm the task landed with the right options.
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redis.Addr()})
	info, err := inspector.ListScheduledTasks(appqueue.QueueCritical, asynq.PageSize(10))
	require.NoError(t, err)
	require.Len(t, info, 1)
	assert.Equal(t, string(appqueue.TypeInterviewScoring), info[0].Type)
}
