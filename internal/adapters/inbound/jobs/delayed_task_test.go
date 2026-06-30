package jobs_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
	"github.com/xcreativs/caliber/internal/adapters/outbound/queue"
)

type noopAsynqLogger struct{}

func (noopAsynqLogger) Debug(_ ...any) {}
func (noopAsynqLogger) Info(_ ...any)  {}
func (noopAsynqLogger) Warn(_ ...any)  {}
func (noopAsynqLogger) Error(_ ...any) {}
func (noopAsynqLogger) Fatal(_ ...any) {}

func TestDelayedTaskFiresOnTime(t *testing.T) {
	redis := miniredis.RunT(t)
	redisOpt := asynq.RedisClientOpt{Addr: redis.Addr()}

	client := asynq.NewClient(redisOpt)
	defer func() { _ = client.Close() }()

	processed := make(chan string, 1)
	mux := jobs.NewMux(
		slog.New(slog.DiscardHandler),
		jobs.WithHealthcheckCallback(func(_ context.Context, payload jobs.HealthcheckPayload) error {
			processed <- payload.Probe
			return nil
		}),
	)

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:              1,
		Queues:                   queue.Priorities(),
		DelayedTaskCheckInterval: 50 * time.Millisecond,
		TaskCheckInterval:        50 * time.Millisecond,
		ShutdownTimeout:          time.Second,
		Logger:                   noopAsynqLogger{},
	})

	done := make(chan error, 1)
	go func() { done <- srv.Start(mux) }()
	t.Cleanup(func() {
		srv.Stop()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("server did not stop")
		}
	})

	payload, err := jobs.EncodeHealthcheckPayload(jobs.HealthcheckPayload{Probe: "delayed"})
	require.NoError(t, err)

	const delay = 200 * time.Millisecond
	info, err := client.EnqueueContext(
		t.Context(),
		asynq.NewTask(jobs.TypeHealthcheck, payload),
		asynq.Queue("default"),
		asynq.ProcessIn(delay),
		asynq.MaxRetry(1),
		asynq.Timeout(time.Second),
	)
	require.NoError(t, err)
	assert.Equal(t, asynq.TaskStateScheduled, info.State)

	// Confirm the task is sitting in the scheduled queue before it is due.
	inspector := asynq.NewInspector(redisOpt)
	scheduled, err := inspector.ListScheduledTasks("default", asynq.PageSize(10))
	require.NoError(t, err)
	require.Len(t, scheduled, 1)
	assert.Equal(t, jobs.TypeHealthcheck, scheduled[0].Type)

	start := time.Now()
	select {
	case probe := <-processed:
		assert.Equal(t, "delayed", probe)
		elapsed := time.Since(start)
		assert.GreaterOrEqual(t, elapsed, delay, "task fired before its scheduled time")
		assert.Less(t, elapsed, delay+300*time.Millisecond, "task took too long to fire")
	case <-time.After(3 * time.Second):
		t.Fatal("delayed task did not fire on time")
	}
}
