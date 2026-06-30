package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	llmadapter "github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/seed"
)

type handles struct {
	users        *memory.UserRepo
	cands        *memory.CandidateRepo
	profs        *memory.TalentProfileRepo
	roles        *memory.RoleRepo
	interviews   *memory.InterviewRepo
	applications *memory.ApplicationRepo
}

func newRepos() (seed.Repositories, handles) {
	h := handles{
		memory.NewUserRepo(), memory.NewCandidateRepo(), memory.NewTalentProfileRepo(),
		memory.NewRoleRepo(), memory.NewInterviewRepo(), memory.NewApplicationRepo(),
	}
	return seed.Repositories{
		Users: h.users, Candidates: h.cands, Profiles: h.profs, Roles: h.roles,
		Interviews: h.interviews, Applications: h.applications,
	}, h
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

func TestLoad_PreRunsInterviewsWhenLLMProvided(t *testing.T) {
	ctx := context.Background()
	repos, h := newRepos()

	res, err := seed.Load(ctx, repos, authadapter.NewArgon2idHasher(), time.Unix(1700000000, 0), seed.WithPreRunInterviews(llmadapter.NewDev()))
	require.NoError(t, err)
	assert.Equal(t, 2, res.Interviews, "two hand-curated hero interviews are pre-run")

	ama := findUser(ctx, t, h.users, "ama.mensah@example.com")
	kofi := findUser(ctx, t, h.users, "kofi.asante@example.com")
	esi := findUser(ctx, t, h.users, "esi.owusu@example.com")
	yaw := findUser(ctx, t, h.users, "yaw.boateng@example.com")

	assertReportCardStored(ctx, t, h.interviews, ama.ID)
	assertReportCardStored(ctx, t, h.interviews, kofi.ID)
	assertNoInterview(ctx, t, h.interviews, esi.ID)
	assertNoInterview(ctx, t, h.interviews, yaw.ID)
}

func TestLoad_PreSeedsAgentStateWhenLLMAndAppsProvided(t *testing.T) {
	ctx := context.Background()
	repos, h := newRepos()

	res, err := seed.Load(ctx, repos, authadapter.NewArgon2idHasher(), time.Unix(1700000000, 0),
		seed.WithPreRunInterviews(llmadapter.NewDev()),
		seed.WithPreSeededAgentState(llmadapter.NewDev(), h.applications),
	)
	require.NoError(t, err)
	assert.Positive(t, res.Applications, "hand-curated heroes produce pre-seeded applications")

	ama := findUser(ctx, t, h.users, "ama.mensah@example.com")
	apps, total, err := h.applications.ByCandidate(ctx, ama.ID, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Positive(t, total, "Ama has a pre-seeded application")
	for _, ap := range apps {
		assert.Equal(t, agentdom.SourceAgent, ap.Source)
		assert.Equal(t, agentdom.StatusSubmitted, ap.Status)
		assert.NotEmpty(t, ap.TailoredSummary)
	}

	runner := candidateagentapp.NewAgentRunner(
		h.cands, h.profs, h.roles, h.applications, llmadapter.NewDev(),
		candidateagentapp.WithWakeUpInsights(h.interviews, nil),
	)
	view, err := runner.WakeUpView(ctx, ama.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, view.ApplicationsSubmitted, 1, "wake-up view shows the pre-seeded application")
	assert.GreaterOrEqual(t, view.ScreeningsCompleted, 1, "wake-up view shows the pre-run screening")
}

func TestLoad_SkipsPreRunWithoutLLM(t *testing.T) {
	ctx := context.Background()
	repos, h := newRepos()

	res, err := seed.Load(ctx, repos, authadapter.NewArgon2idHasher(), time.Unix(1700000000, 0))
	require.NoError(t, err)
	assert.Zero(t, res.Interviews)

	candidates, _, err := h.cands.List(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	for _, c := range candidates {
		assertNoInterview(ctx, t, h.interviews, c.ID)
	}
}

func TestLoad_SkipsPreSeedWithoutLLMOrApps(t *testing.T) {
	ctx := context.Background()
	repos, h := newRepos()

	// No LLM: pre-seed should be skipped even though an apps repo is available.
	res, err := seed.Load(ctx, repos, authadapter.NewArgon2idHasher(), time.Unix(1700000000, 0))
	require.NoError(t, err)
	assert.Zero(t, res.Applications)

	ama := findUser(ctx, t, h.users, "ama.mensah@example.com")
	_, total, err := h.applications.ByCandidate(ctx, ama.ID, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Zero(t, total, "no applications without pre-seed")
}

func TestDefaultPassword_IsLoginable(t *testing.T) {
	hasher := authadapter.NewArgon2idHasher()
	hash, err := hasher.Hash(seed.DefaultPassword)
	require.NoError(t, err)
	ok, err := hasher.Verify(hash, seed.DefaultPassword)
	require.NoError(t, err)
	assert.True(t, ok, "seeded demo accounts can log in with DefaultPassword")
}

func findUser(ctx context.Context, t *testing.T, users *memory.UserRepo, email string) *identity.User {
	t.Helper()
	u, err := users.ByEmail(ctx, identity.Email(email))
	require.NoErrorf(t, err, "find user %s", email)
	return u
}

func assertReportCardStored(ctx context.Context, t *testing.T, interviews *memory.InterviewRepo, candidateID kernel.ID) {
	t.Helper()
	ivs, total, err := interviews.ByCandidate(ctx, candidateID, kernel.NewPage(1, 100))
	require.NoError(t, err)
	require.Equalf(t, int64(1), total, "candidate %s should have one pre-run interview", candidateID)
	require.NotNil(t, ivs[0].Report, "pre-run interview should have a report card")
	assert.NotEqual(t, interviewdom.VerdictUnspecified, ivs[0].Report.Verdict)
	assert.NotEmpty(t, ivs[0].Report.Scores, "report card has scored competencies")
}

func assertNoInterview(ctx context.Context, t *testing.T, interviews *memory.InterviewRepo, candidateID kernel.ID) {
	t.Helper()
	ivs, total, err := interviews.ByCandidate(ctx, candidateID, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.Zerof(t, total, "candidate %s should not have a pre-run interview", candidateID)
	assert.Empty(t, ivs)
}
