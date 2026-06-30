package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

func TestUserRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewUserRepo()
	u, err := identity.NewUser(identity.Email("a@b.com"), identity.RoleCandidate, "A", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, r.Create(ctx, u))

	r.Reset()

	_, err = r.ByID(ctx, u.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	_, err = r.ByEmail(ctx, u.Email)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestRoleRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewRoleRepo()
	employerID := kernel.NewID()
	rl, err := role.NewRole(employerID, role.RoleSpec{
		Title:      "T",
		Seniority:  role.SeniorityMid,
		SalaryBand: kernel.SalaryBand{Currency: "GHS", Low: 1000, High: 2000},
	}, role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1}}}, time.Now())
	require.NoError(t, err)
	require.NoError(t, r.Create(ctx, rl))

	r.Reset()

	_, err = r.ByID(ctx, rl.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestCandidateRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewCandidateRepo()
	c, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NoError(t, r.Create(ctx, c))

	r.Reset()

	_, err = r.ByID(ctx, c.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestTalentProfileRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewTalentProfileRepo()
	p, err := talent.NewTalentProfile(kernel.NewID(), "summary", nil)
	require.NoError(t, err)
	require.NoError(t, r.Create(ctx, p))

	r.Reset()

	_, err = r.ByID(ctx, p.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestApplicationRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewApplicationRepo()
	app, err := candidateagent.NewAgentApplication(kernel.NewID(), kernel.NewID(), kernel.NewID(), "summary")
	require.NoError(t, err)
	require.NoError(t, r.Create(ctx, app))

	r.Reset()

	_, err = r.ByID(ctx, app.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestInterviewRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewInterviewRepo()
	iv := &interview.Interview{
		ID:          kernel.NewID(),
		RoleID:      kernel.NewID(),
		CandidateID: kernel.NewID(),
	}
	require.NoError(t, r.Create(ctx, iv))

	r.Reset()

	_, err := r.ByID(ctx, iv.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestMatchRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewMatchRepo()
	m := &matching.Match{
		ID:          kernel.NewID(),
		RoleID:      kernel.NewID(),
		CandidateID: kernel.NewID(),
	}
	require.NoError(t, r.Upsert(ctx, m))

	r.Reset()

	list, total, err := r.ByRole(ctx, m.RoleID, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)
}

func TestRefreshStoreReset(t *testing.T) {
	ctx := context.Background()
	s := memory.NewRefreshStore()
	require.NoError(t, s.Save(ctx, app.RefreshRecord{
		ID:        "jti",
		UserID:    kernel.NewID(),
		ExpiresAt: time.Now().Add(time.Hour),
	}))

	s.Reset()

	_, err := s.Consume(ctx, "jti", time.Now())
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestAuditRepoReset(t *testing.T) {
	ctx := context.Background()
	r := memory.NewAuditRepo()
	require.NoError(t, r.Append(ctx, &audit.AuditEntry{
		ID:          kernel.NewID(),
		ActorUserID: kernel.NewID(),
		Action:      "test",
		Entity:      "user",
		EntityID:    kernel.NewID(),
		Timestamp:   time.Now(),
	}))

	r.Reset()

	list, total, err := r.List(ctx, "user", kernel.NewID(), kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)
}
