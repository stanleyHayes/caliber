package jobs_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
	"github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
)

func TestFailingTaskLandsInArchiveAfterMaxRetry(t *testing.T) {
	redis := miniredis.RunT(t)
	redisOpt := asynq.RedisClientOpt{Addr: redis.Addr()}

	logs := &lockedBuffer{}
	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mux := jobs.NewMux(
		logger,
		jobs.WithHealthcheckCallback(func(context.Context, jobs.HealthcheckPayload) error {
			return errors.New("poison task")
		}),
	)

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:              1,
		Queues:                   queue.Priorities(),
		RetryDelayFunc:           func(int, error, *asynq.Task) time.Duration { return 10 * time.Millisecond },
		ErrorHandler:             jobs.NewArchiveAlertHandler(logger),
		DelayedTaskCheckInterval: 20 * time.Millisecond,
		TaskCheckInterval:        20 * time.Millisecond,
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

	client := asynq.NewClient(redisOpt)
	defer func() { _ = client.Close() }()
	payload, err := jobs.EncodeHealthcheckPayload(jobs.HealthcheckPayload{Probe: "archive"})
	require.NoError(t, err)
	_, err = client.EnqueueContext(
		t.Context(),
		asynq.NewTask(jobs.TypeHealthcheck, payload),
		asynq.Queue(appqueue.QueueDefault),
		asynq.MaxRetry(1),
		asynq.Timeout(time.Second),
	)
	require.NoError(t, err)

	inspector := jobs.NewArchiveInspector(redisOpt)
	defer func() { require.NoError(t, inspector.Close()) }()

	var archived []*asynq.TaskInfo
	var listErr error
	require.Eventually(t, func() bool {
		redis.FastForward(25 * time.Millisecond)
		archived, listErr = inspector.ListArchived(t.Context(), appqueue.QueueDefault, asynq.PageSize(10))
		return listErr == nil && len(archived) == 1
	}, 3*time.Second, 20*time.Millisecond)
	require.NoError(t, listErr)
	require.Len(t, archived, 1)
	assert.Equal(t, jobs.TypeHealthcheck, archived[0].Type)
	assert.Equal(t, 1, archived[0].MaxRetry)

	assert.Eventually(t, func() bool {
		got := logs.String()
		return strings.Contains(got, "task archived after max retries") &&
			strings.Contains(got, "dead_letter") &&
			strings.Contains(got, jobs.TypeHealthcheck)
	}, time.Second, 10*time.Millisecond)
}

type lockedBuffer struct {
	bytes.Buffer

	mu sync.Mutex
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}
