package wiring_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/wiring"
)

func TestOpenRepositoriesInMemoryPath(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{SeedDemo: false}
	repos, cleanup, checks, err := wiring.OpenRepositories(ctx, cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	defer cleanup()

	assert.NotNil(t, repos.Roles)
	assert.NotNil(t, repos.Users)
	assert.NotNil(t, repos.Candidates)
	assert.Nil(t, repos.Pool)
	assert.Empty(t, checks)
}

func TestOpenRepositoriesFailsWithBadDatabaseURL(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DatabaseURL: "postgres://invalid", SeedDemo: false}
	_, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, slog.New(slog.DiscardHandler))
	defer cleanup()
	require.Error(t, err)
}

func TestOpenRepositoriesFailsWhenPingFails(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DatabaseURL: "postgres://localhost:1/caliber?sslmode=disable&connect_timeout=1", SeedDemo: false}
	_, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, slog.New(slog.DiscardHandler))
	defer cleanup()
	require.Error(t, err)
}

func TestSeedDemoLoadsDemoDataset(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{SeedDemo: true}
	repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	defer cleanup()

	roles, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.NotEmpty(t, roles)
}

func TestBuildLLMDevPath(t *testing.T) {
	cfg := config.Config{}
	llmClient, recorder := wiring.BuildLLM(cfg, slog.New(slog.DiscardHandler), nil)
	assert.NotNil(t, llmClient)
	assert.NotNil(t, recorder)
}

func TestBuildLLMClaudePath(t *testing.T) {
	cfg := config.Config{AnthropicAPIKey: "sk-test", AnthropicModel: "claude-3-5-sonnet"}
	llmClient, recorder := wiring.BuildLLM(cfg, slog.New(slog.DiscardHandler), nil)
	assert.NotNil(t, llmClient)
	assert.NotNil(t, recorder)
}

func TestBuildEmbedderDevPath(t *testing.T) {
	cfg := config.Config{}
	embedder := wiring.BuildEmbedder(cfg, slog.New(slog.DiscardHandler))
	assert.NotNil(t, embedder)
}

func TestBuildEmbedderOpenAIPath(t *testing.T) {
	cfg := config.Config{OpenAIAPIKey: "sk-test", OpenAIEmbeddingModel: "text-embedding-3-small"}
	embedder := wiring.BuildEmbedder(cfg, slog.New(slog.DiscardHandler))
	assert.NotNil(t, embedder)
}

func TestSeedGeneratedLoadsGeneratedDataset(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{SeedDemo: true, SeedGenerated: true}
	repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	defer cleanup()

	roles, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(roles), 8)

	candidates, total, err := repos.Candidates.List(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(50))
	assert.Len(t, candidates, len(candidates))
}

func TestResetRepositoriesWipesInMemoryData(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{SeedDemo: true}
	repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, wiring.ResetRepositories(ctx, repos))

	roles, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestReseedReloadsDemoDataset(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{SeedDemo: true}
	require.NoError(t, wiring.Reseed(ctx, cfg, slog.New(slog.DiscardHandler)))
}

func TestNewAsynqmonHandlerReturnsNilForEmptyRedisURL(t *testing.T) {
	h, cleanup, err := wiring.NewAsynqmonHandler("")
	require.NoError(t, err)
	assert.Nil(t, h)
	cleanup()
}

func TestNewAsynqmonHandlerReturnsErrorForInvalidRedisURL(t *testing.T) {
	_, cleanup, err := wiring.NewAsynqmonHandler("://not-a-url")
	require.Error(t, err)
	cleanup()
}
