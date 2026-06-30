package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	llmadapter "github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/seed"
)

func TestGenerator_GeneratesParserAcceptedDataset(t *testing.T) {
	ctx := context.Background()
	repos, h := newRepos()
	gen := seed.NewGenerator(
		authadapter.NewArgon2idHasher(),
		llmadapter.NewDev(),
		func() time.Time { return time.Unix(1700000000, 0) },
	)

	res, err := gen.Generate(ctx, repos)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, res.Employers, 6)
	assert.LessOrEqual(t, res.Employers, 8)
	assert.GreaterOrEqual(t, res.Roles, 8)
	assert.LessOrEqual(t, res.Roles, 12)
	assert.GreaterOrEqual(t, res.Candidates, 50)
	assert.LessOrEqual(t, res.Candidates, 60)

	candidates, total, err := h.cands.List(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Equal(t, int64(res.Candidates), total)

	for _, c := range candidates {
		assert.Equal(t, c.UserID, c.ID, "provisioning convention: candidate.ID == user.ID")
		profile, perr := h.profs.ByCandidateID(ctx, c.ID)
		require.NoErrorf(t, perr, "profile exists for candidate %s", c.ID)
		require.NotEmpty(t, profile.Competencies, "parser produced at least one competency")
		for _, comp := range profile.Competencies {
			assert.NotEmpty(t, comp.EvidenceQuote, "no-fabrication: every competency traces to CV evidence")
			assert.NotEmpty(t, comp.SourceSpan, "no-fabrication: every competency has a source span")
		}
		assert.Equal(t, "screened", profile.PassportStatus.String(), "generated profiles are demo-screened")
	}

	roles, totalRoles, err := h.roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Equal(t, int64(res.Roles), totalRoles)
	for _, rl := range roles {
		assert.NotEmpty(t, rl.Spec.Title, "generated role has a title")
		assert.True(t, rl.Spec.Seniority.Valid(), "generated role has a valid seniority")
		assert.InDelta(t, 1.0, rl.Rubric.TotalWeight(), 0.01, "generated role rubric weights are normalised")
	}
}

func TestGenerator_DeterministicAcrossRuns(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }
	gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), llmadapter.NewDev(), now)

	repos1, _ := newRepos()
	res1, err := gen.Generate(ctx, repos1)
	require.NoError(t, err)

	repos2, _ := newRepos()
	res2, err := gen.Generate(ctx, repos2)
	require.NoError(t, err)

	assert.Equal(t, res1, res2, "generator is deterministic")
}
