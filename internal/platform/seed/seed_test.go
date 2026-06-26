package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/seed"
)

type handles struct {
	users *memory.UserRepo
	cands *memory.CandidateRepo
	profs *memory.TalentProfileRepo
	roles *memory.RoleRepo
}

func newRepos() (seed.Repositories, handles) {
	h := handles{memory.NewUserRepo(), memory.NewCandidateRepo(), memory.NewTalentProfileRepo(), memory.NewRoleRepo()}
	return seed.Repositories{Users: h.users, Candidates: h.cands, Profiles: h.profs, Roles: h.roles}, h
}

func TestLoad_PopulatesConsistentDataset(t *testing.T) {
	repos, h := newRepos()
	res, err := seed.Load(context.Background(), repos, authadapter.NewArgon2idHasher(), time.Unix(1700000000, 0))
	require.NoError(t, err)
	assert.Equal(t, 3, res.Employers)
	assert.Equal(t, 5, res.Roles)
	assert.Equal(t, 8, res.Candidates)

	candidates, total, err := h.cands.List(context.Background(), kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Equal(t, int64(8), total)
	for _, c := range candidates {
		assert.Equal(t, c.UserID, c.ID, "provisioning convention: candidate.ID == user.ID")
		p, perr := h.profs.ByCandidateID(context.Background(), c.ID)
		require.NoErrorf(t, perr, "profile exists for %s", c.ID)
		assert.NotEmpty(t, p.Competencies)
	}
}

func TestLoad_ProducesTwoWayAlerts(t *testing.T) {
	repos, h := newRepos()
	_, err := seed.Load(context.Background(), repos, authadapter.NewArgon2idHasher(), time.Unix(1700000000, 0))
	require.NoError(t, err)

	// The seeded data must be "alive": the Radar alert feed surfaces strong
	// two-way matches (e.g. Ama -> Senior Backend, Yaw -> Platform).
	agg := dashboardapp.NewAggregator(h.cands, h.profs, h.users, h.roles)
	alerts, totalAlerts, err := agg.Alerts(context.Background(), kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Positive(t, totalAlerts, "demo data produces two-way match alerts")
	assert.NotEmpty(t, alerts)
}

func TestDefaultPassword_IsLoginable(t *testing.T) {
	hasher := authadapter.NewArgon2idHasher()
	hash, err := hasher.Hash(seed.DefaultPassword)
	require.NoError(t, err)
	ok, err := hasher.Verify(hash, seed.DefaultPassword)
	require.NoError(t, err)
	assert.True(t, ok, "seeded demo accounts can log in with DefaultPassword")
}
