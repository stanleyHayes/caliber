package wiring_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/wiring"
)

func TestResetRepositoriesClearsInMemoryRepos(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{SeedDemo: true}
	repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	defer cleanup()

	// Precondition: seeding populated the in-memory stores.
	rolesBefore, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	require.NotEmpty(t, rolesBefore)
	candsBefore, _, err := repos.Candidates.List(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	require.NotEmpty(t, candsBefore)

	require.NoError(t, wiring.ResetRepositories(ctx, repos))

	rolesAfter, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Empty(t, rolesAfter)
	candsAfter, total, err := repos.Candidates.List(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, candsAfter)
}

func TestReseedDeterministicAndRepeatable(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.DiscardHandler)
	cfg := config.Config{SeedDemo: true}

	var roles1, cands1 int
	for i := range 2 {
		require.NoError(t, wiring.Reseed(ctx, cfg, log))
		repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, log)
		require.NoError(t, err)
		defer cleanup()
		roles, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
		require.NoError(t, err)
		cands, total, err := repos.Candidates.List(ctx, kernel.NewPage(1, 100))
		require.NoError(t, err)
		if i == 0 {
			roles1 = len(roles)
			cands1 = int(total)
			require.Positive(t, roles1)
			require.Positive(t, cands1)
			require.Len(t, cands, cands1)
			continue
		}
		assert.Len(t, roles, roles1, "reseed produces deterministic role counts")
		assert.Equal(t, cands1, int(total), "reseed produces deterministic candidate counts")
	}
}

func TestReseedWipesExistingData(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.DiscardHandler)
	cfg := config.Config{SeedDemo: true}

	// First seed.
	require.NoError(t, wiring.Reseed(ctx, cfg, log))
	repos1, cleanup1, _, err := wiring.OpenRepositories(ctx, cfg, log)
	require.NoError(t, err)
	defer cleanup1()
	roles1, _, err := repos1.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	require.NotEmpty(t, roles1)

	// Add an extra candidate, then reseed to confirm it is wiped.
	extra, err := createExtraCandidate(ctx, repos1)
	require.NoError(t, err)
	require.NoError(t, wiring.Reseed(ctx, cfg, log))

	repos2, cleanup2, _, err := wiring.OpenRepositories(ctx, cfg, log)
	require.NoError(t, err)
	defer cleanup2()

	_, err = repos2.Candidates.ByID(ctx, extra.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	roles2, _, err := repos2.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Len(t, roles2, len(roles1), "reseed restores the baseline dataset")
}

func createExtraCandidate(ctx context.Context, repos wiring.Repositories) (*talent.Candidate, error) {
	email, err := identity.NewEmail("extra.reset@example.com")
	if err != nil {
		return nil, err
	}
	u, err := identity.NewUser(email, identity.RoleCandidate, "Extra Reset", "hash", time.Now())
	if err != nil {
		return nil, err
	}
	if err := repos.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	c, err := talent.NewCandidate(u.ID, "Accra", talent.CandidateIntake{})
	if err != nil {
		return nil, err
	}
	c.ID = u.ID
	return c, repos.Candidates.Create(ctx, c)
}
