package grpcadapter

// Chaos & resilience tests at the inbound gRPC boundary.
//
// These assert that when an outbound dependency fails — the Asynq/Redis queue on
// enqueue, or the Postgres repository on a read/write — the gRPC handler degrades
// cleanly: it returns a typed status error to the client (never a silent success,
// never a panic) and loses no data silently. They also pin the REAL error-to-code
// mapping so a reviewer can see exactly what a client observes when a core
// dependency is down.
//
// NOTE ON codes.Unavailable: the kernel deliberately has no "Unavailable" kind.
// Infrastructure faults (a closed pool, a dead Redis) arrive as unclassified
// errors and map to codes.Internal with an opaque message, by design, so raw
// pgx/redis detail never leaks to a client (CWE-209). These tests assert that
// real behaviour rather than an aspirational Unavailable. See docs/chaos-testing.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

// erroringDispatcher is a TaskDispatcher whose enqueue always fails, standing in
// for a down Redis / unreachable Asynq broker. It is NOT a *Noop, so the handler
// takes the enqueue path (not the synchronous dev fallback).
type erroringDispatcher struct{ err error }

func (d erroringDispatcher) DispatchCandidateAgentRun(context.Context, kernel.ID, ...appqueue.DispatchOption) (string, error) {
	return "", d.err
}

func (d erroringDispatcher) DispatchInterviewScoring(context.Context, kernel.ID, ...appqueue.DispatchOption) (string, error) {
	return "", d.err
}

func (d erroringDispatcher) DispatchBatchRematch(context.Context, kernel.ID, ...appqueue.DispatchOption) (string, error) {
	return "", d.err
}

func (d erroringDispatcher) Close() error { return nil }

// TestRunAgentQueueDownReturnsCleanError proves that when the queue enqueue fails
// (Redis unreachable), RunAgent surfaces a typed gRPC error to the client instead
// of returning a bogus success — no job is silently dropped without the caller
// being told. The wrapped enqueue error is unclassified, so it maps to Internal.
func TestRunAgentQueueDownReturnsCleanError(t *testing.T) {
	cid := kernel.NewID()
	// A non-Noop dispatcher whose enqueue fails, mimicking asynq.EnqueueContext
	// wrapping a redis connection error.
	dispatcher := erroringDispatcher{err: errors.New("queue: enqueue candidate_agent:run: dial tcp 127.0.0.1:6379: connect: connection refused")}
	srv := NewAgentServer(nil, nil, dispatcher)

	resp, err := srv.RunAgent(asCandidate(context.Background(), cid), &caliberv1.RunAgentRequest{CandidateId: cid.String()})
	require.Error(t, err, "a failed enqueue must be reported to the client, not swallowed")
	assert.Nil(t, resp, "no job id is returned when nothing was enqueued")
	assert.Equal(t, codes.Internal, status.Code(err),
		"an unclassified queue/redis failure maps to codes.Internal (opaque, CWE-209)")
	assert.Equal(t, "internal error", status.Convert(err).Message(),
		"the raw redis dial detail must not leak to the client")
}

// TestTimeAdvanceQueueDownReturnsCleanError proves the same graceful degradation
// on the demo "overnight" action: a down queue yields a typed error, no panic.
func TestTimeAdvanceQueueDownReturnsCleanError(t *testing.T) {
	cid := kernel.NewID()
	dispatcher := erroringDispatcher{err: errors.New("queue: enqueue candidate_agent:run: redis: connection pool timeout")}
	srv := NewAgentServer(nil, nil, dispatcher)

	resp, err := srv.TimeAdvance(asCandidate(context.Background(), cid), &caliberv1.TimeAdvanceRequest{CandidateId: cid.String()})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// TestRunAgentDBDownReturnsCleanError proves that in the synchronous (Noop
// dispatcher / dev) path, a DB repository failure during the agent run surfaces
// as a clean gRPC error rather than a panic. The candidate lookup fails with a
// raw infrastructure error (a stand-in for a closed pgx pool), which the mapper
// renders as an opaque Internal.
func TestRunAgentDBDownReturnsCleanError(t *testing.T) {
	ctrl := gomock.NewController(t)
	cid := kernel.NewID()
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	roles := mocks.NewMockRoleRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	apps := memory.NewApplicationRepo()

	// The pool is "down": the candidate lookup returns a raw (non-kernel) error.
	candidates.EXPECT().ByID(gomock.Any(), cid).
		Return(nil, errors.New("failed to connect to `host=db`: dial error: connection refused")).AnyTimes()

	runner := candidateagentapp.NewAgentRunner(candidates, profiles, roles, apps, llm)
	// nil dispatcher -> Noop -> synchronous path that actually touches the repo.
	srv := NewAgentServer(runner, apps, nil)

	resp, err := srv.RunAgent(asCandidate(context.Background(), cid), &caliberv1.RunAgentRequest{CandidateId: cid.String()})
	require.Error(t, err, "a DB failure must surface, never panic")
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err),
		"an unclassified DB failure maps to codes.Internal")
	assert.Equal(t, "internal error", status.Convert(err).Message(),
		"raw pgx connection detail must not reach the client")
}

// TestErrToStatusMapsEveryKind pins the full domain-kind -> gRPC-code contract the
// resilience story depends on: kernel errors carry their meaning to the client,
// while any UNCLASSIFIED infrastructure error (DB/Redis/LLM transport down)
// collapses to an opaque Internal. This is the single source of truth a reviewer
// can check the "clean fallbacks" claim against.
func TestErrToStatusMapsEveryKind(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		want    codes.Code
		message string // "" means: don't assert the message
	}{
		{"invalid", kernel.Invalid("bad input"), codes.InvalidArgument, "bad input"},
		{"not_found", kernel.NotFound("missing"), codes.NotFound, "missing"},
		{"conflict", kernel.Conflict("dupe"), codes.AlreadyExists, "dupe"},
		{"unauthorized", kernel.Unauthorized("no auth"), codes.Unauthenticated, "no auth"},
		{"forbidden", kernel.Forbidden("nope"), codes.PermissionDenied, "nope"},
		{"rate_limited", kernel.TooManyRequests("slow down"), codes.ResourceExhausted, "slow down"},
		// The infrastructure-down cases: all opaque Internal, no raw detail leaked.
		{"db_pool_closed", errors.New("closed pool"), codes.Internal, "internal error"},
		{"redis_down", errors.New("dial tcp :6379: connection refused"), codes.Internal, "internal error"},
		{"llm_transport", kernel.Wrap(errors.New("provider down"), kernel.KindInternal, "llm: model call failed"), codes.Internal, "internal error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := errToStatus(tc.err)
			assert.Equal(t, tc.want, status.Code(out))
			if tc.message != "" {
				assert.Equal(t, tc.message, status.Convert(out).Message())
			}
		})
	}
}
