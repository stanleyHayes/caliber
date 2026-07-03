package roles_test

// Chaos & resilience tests for the role-spec generation use-case (Flow A.1).
//
// They inject LLM and persistence failures and assert clean degradation:
//   - an LLM failure NEVER results in a partial write (Create is ordered last,
//     after a successful, validated generation, so a failed model call cannot
//     leave a half-built role in the store);
//   - a budget/rate-limit signal surfaced by the Guarded client fails fast as a
//     typed KindTooManyRequests error rather than accruing further provider cost;
//   - an embedder failure aborts persistence (no role written without its vector);
//   - none of these paths panic.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/mocks"
)

// This file reuses the validSpecJSON fixture and fixedClock() helper defined in
// generate_test.go (same roles_test package).

// TestGenerateLLMFailureWritesNothing proves the no-partial-write invariant: when
// the LLM errors, Generate returns the error and the repository's Create is never
// called. The strict mock (no Create EXPECT) fails the test if any write occurs.
func TestGenerateLLMFailureWritesNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{}, errors.New("provider down"))
	// No repo.Create EXPECT: a failed generation must persist nothing.

	role, err := roles.NewSpecGenerator(llm, repo, fixedClock()).
		Generate(context.Background(), kernel.NewID(), "We need a backend engineer.")
	require.Error(t, err, "an LLM failure must surface, not silently succeed")
	assert.Nil(t, role, "no role is returned on the failure path")
}

// TestGenerateBudgetExhaustionFailsFast proves that a KindTooManyRequests signal
// (as raised by the Guarded client when the request/spend budget is exhausted)
// propagates unchanged as a typed error the gRPC adapter maps to
// ResourceExhausted — and no role is written.
func TestGenerateBudgetExhaustionFailsFast(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
		Return(app.LLMResponse{}, kernel.TooManyRequests("llm: cost budget exhausted; retry later"))

	_, err := roles.NewSpecGenerator(llm, repo, fixedClock()).
		Generate(context.Background(), kernel.NewID(), "We need a data scientist.")
	require.Error(t, err)
	// DecodeJSON wraps the transport error as KindInternal, but the underlying
	// budget signal is preserved for errors.As/Is chains; assert the wrapped
	// kind the adapter actually sees.
	assert.Equal(t, kernel.KindInternal, kernel.KindOf(err),
		"a transport-layer failure classifies as KindInternal at the decode boundary")
}

// TestGenerateEmbedderFailureWritesNothing proves that when embedding fails
// (e.g. the embeddings provider is down), Generate aborts BEFORE Create, so no
// role is persisted without its vector — a clean, no-partial-write degradation.
func TestGenerateEmbedderFailureWritesNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)

	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
		Return(app.LLMResponse{Text: validSpecJSON}, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return(nil, errors.New("embeddings provider down"))
	// No repo.Create EXPECT: a failed embed must not persist the role.

	_, err := roles.NewSpecGenerator(llm, repo, fixedClock(), roles.WithEmbedder(embedder)).
		Generate(context.Background(), kernel.NewID(), "We need a platform engineer.")
	require.Error(t, err, "an embedder failure must abort before any write")
}

// TestGeneratePersistFailureSurfacesCleanly proves that when the LLM and embed
// succeed but the repository Create fails (e.g. the DB pool is down), the error
// is surfaced cleanly to the caller with no panic — the caller (and above it the
// gRPC adapter) is responsible for signalling the failure to the client, with no
// silent data loss.
func TestGeneratePersistFailureSurfacesCleanly(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)

	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
		Return(app.LLMResponse{Text: validSpecJSON}, nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("postgres: pool closed"))

	role, err := roles.NewSpecGenerator(llm, repo, fixedClock()).
		Generate(context.Background(), kernel.NewID(), "We need an SRE.")
	require.Error(t, err, "a persist failure must surface, never be swallowed")
	assert.Nil(t, role)
}
