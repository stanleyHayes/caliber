package jobs_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
	"github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestHealthcheckTaskProcessesDirectly(t *testing.T) {
	payload, err := jobs.EncodeHealthcheckPayload(jobs.HealthcheckPayload{Probe: "direct"})
	require.NoError(t, err)
	seen := make(chan jobs.HealthcheckPayload, 1)
	mux := jobs.NewMux(slog.New(slog.DiscardHandler), jobs.WithHealthcheckCallback(func(_ context.Context, payload jobs.HealthcheckPayload) error {
		seen <- payload
		return nil
	}))

	err = mux.ProcessTask(t.Context(), asynqTask(jobs.TypeHealthcheck, payload))

	require.NoError(t, err)
	assert.Equal(t, "direct", (<-seen).Probe)
}

func asynqTask(taskType string, payload []byte) *asynq.Task {
	return asynq.NewTask(taskType, payload)
}

func TestEnqueueToProcessRoundTrip(t *testing.T) {
	redis := miniredis.RunT(t)
	redisURL := "redis://" + redis.Addr() + "/0"
	redisOpt, err := queue.RedisOpt(redisURL)
	require.NoError(t, err)
	client := asynq.NewClient(redisOpt)
	defer func() { require.NoError(t, client.Close()) }()

	processed := make(chan jobs.HealthcheckPayload, 1)
	worker, err := jobs.NewWorker(redisURL, slog.New(slog.DiscardHandler), jobs.WithHealthcheckCallback(
		func(_ context.Context, payload jobs.HealthcheckPayload) error {
			processed <- payload
			return nil
		},
	))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("worker did not stop")
		}
	})

	payload, err := jobs.EncodeHealthcheckPayload(jobs.HealthcheckPayload{Probe: "round-trip"})
	require.NoError(t, err)
	enqueued, err := client.EnqueueContext(
		t.Context(),
		asynq.NewTask(jobs.TypeHealthcheck, payload),
		asynq.Queue("default"),
		asynq.MaxRetry(1),
		asynq.Timeout(time.Second),
	)
	require.NoError(t, err)
	assert.Equal(t, jobs.TypeHealthcheck, enqueued.Type)
	assert.Equal(t, "default", enqueued.Queue)

	select {
	case got := <-processed:
		assert.Equal(t, "round-trip", got.Probe)
	case <-time.After(5 * time.Second):
		t.Fatal("task was not processed")
	}
}

func TestBusinessHandlersReturnClearErrorsWhenDepsMissing(t *testing.T) {
	mux := jobs.NewMux(slog.New(slog.DiscardHandler))
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{}, slog.New(slog.DiscardHandler))

	candidatePayload := mustJSON(t, appqueue.CandidateAgentRunPayload{CandidateID: kernel.NewID().String()})
	err := mux.ProcessTask(t.Context(), asynqTask(string(appqueue.TypeCandidateAgentRun), candidatePayload))
	require.ErrorContains(t, err, "candidate agent runner not wired")

	interviewPayload := mustJSON(t, appqueue.InterviewScoringPayload{InterviewID: kernel.NewID().String()})
	err = mux.ProcessTask(t.Context(), asynqTask(string(appqueue.TypeInterviewScoring), interviewPayload))
	require.ErrorContains(t, err, "interviewer not wired")
}

func TestBatchRematchHandlerValidatesPayload(t *testing.T) {
	mux := jobs.NewMux(slog.New(slog.DiscardHandler))
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{}, slog.New(slog.DiscardHandler))

	payload := mustJSON(t, appqueue.BatchRematchPayload{RoleID: kernel.NewID().String()})
	err := mux.ProcessTask(t.Context(), asynqTask(string(appqueue.TypeBatchRematch), payload))
	require.NoError(t, err)

	payload = mustJSON(t, appqueue.BatchRematchPayload{})
	err = mux.ProcessTask(t.Context(), asynqTask(string(appqueue.TypeBatchRematch), payload))
	require.ErrorContains(t, err, "invalid role_id")
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
