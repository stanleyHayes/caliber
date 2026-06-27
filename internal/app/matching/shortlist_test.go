package matching_test

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	score06  = `{"overall_score":0.6,"confidence":"medium","breakdown":[{"competency":"Go","score":3,"evidence":"x"}],"rationale":"solid","watch_outs":[],"thin_evidence":false}`
	score09  = `{"overall_score":0.9,"confidence":"high","breakdown":[{"competency":"Go","score":4.5,"evidence":"y"}],"rationale":"excellent","watch_outs":["thin mentoring"],"thin_evidence":false}`
	scoreGo1 = `{"overall_score":0.3,"confidence":"low","breakdown":[{"competency":"Go","score":1,"evidence":"z"}],"rationale":"weak","watch_outs":[],"thin_evidence":true}`
)

// shortDeps bundles the mock collaborators of the Shortlister for terse setup.
type shortDeps struct {
	roles      *mocks.MockRoleRepository
	candidates *mocks.MockCandidateRepository
	profiles   *mocks.MockTalentProfileRepository
	recaller   *mocks.MockCandidateRecaller
	embedder   *mocks.MockEmbedder
	scorer     *mocks.MockLLMClient
	matchRepo  *mocks.MockMatchRepository
}

func newDeps(ctrl *gomock.Controller) shortDeps {
	return shortDeps{
		roles:      mocks.NewMockRoleRepository(ctrl),
		candidates: mocks.NewMockCandidateRepository(ctrl),
		profiles:   mocks.NewMockTalentProfileRepository(ctrl),
		recaller:   mocks.NewMockCandidateRecaller(ctrl),
		embedder:   mocks.NewMockEmbedder(ctrl),
		scorer:     mocks.NewMockLLMClient(ctrl),
		matchRepo:  mocks.NewMockMatchRepository(ctrl),
	}
}

func (d shortDeps) shortlister() *matchingapp.Shortlister {
	return matchingapp.NewShortlister(d.roles, d.candidates, d.profiles, d.recaller, d.embedder, d.scorer, d.matchRepo)
}

func validRole(t *testing.T) *role.Role {
	t.Helper()
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid,
			Responsibilities: []string{"build services"}, MustHaves: []string{"Go"},
		},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	return rl
}

func candidateAt(t *testing.T, location string) *talent.Candidate {
	t.Helper()
	c, err := talent.NewCandidate(kernel.NewID(), location, talent.CandidateIntake{})
	require.NoError(t, err)
	return c
}

func profileFor(t *testing.T, cid kernel.ID) *talent.TalentProfile {
	t.Helper()
	p, err := talent.NewTalentProfile(cid, "summary",
		[]talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "built services"}})
	require.NoError(t, err)
	return p
}

func TestGenerateShortlistRanksAndPersists(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := validRole(t)
	c1, c2 := kernel.NewID(), kernel.NewID()

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1, 0.2}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{c1, c2}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), c1).Return(candidateAt(t, "Accra"), nil)
	d.candidates.EXPECT().ByID(gomock.Any(), c2).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), c1).Return(profileFor(t, c1), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), c2).Return(profileFor(t, c2), nil)
	gomock.InOrder(
		d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score06}, nil),
		d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score09}, nil),
	)
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	res, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)
	require.Len(t, res.Matches, 2)
	assert.Equal(t, 2, res.PoolDepth, "pool depth equals the strong-match total")
	assert.Empty(t, res.Exclusions)
	assert.Equal(t, c2, res.Matches[0].CandidateID, "candidate scored 0.9 ranks first")
	assert.InDelta(t, 0.9, res.Matches[0].OverallScore, 1e-9)
	assert.Equal(t, kernel.ConfidenceHigh, res.Matches[0].Confidence)
	assert.Equal(t, c1, res.Matches[1].CandidateID)
	assert.InDelta(t, 0.6, res.Matches[1].OverallScore, 1e-9)
}

// TestGenerateShortlistHardFilters proves stage-3 gating: a location-mismatched
// candidate is excluded BEFORE scoring (no LLM call), an under-scored must-have
// candidate is excluded AFTER scoring, and only the qualifying candidate is
// persisted and returned — each exclusion carrying a reason.
func TestGenerateShortlistHardFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := validRole(t)
	cLagos, cWeak, cGood := kernel.NewID(), kernel.NewID(), kernel.NewID()

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{cLagos, cWeak, cGood}, nil)

	d.candidates.EXPECT().ByID(gomock.Any(), cLagos).Return(candidateAt(t, "Lagos"), nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cWeak).Return(candidateAt(t, "Accra"), nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cGood).Return(candidateAt(t, "Accra"), nil)

	// cLagos is gated out pre-scoring: its profile is never loaded, never scored.
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cWeak).Return(profileFor(t, cWeak), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cGood).Return(profileFor(t, cGood), nil)
	gomock.InOrder(
		d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: scoreGo1}, nil),
		d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score09}, nil),
	)
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	res, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 25)
	require.NoError(t, err)

	require.Len(t, res.Matches, 1)
	assert.Equal(t, cGood, res.Matches[0].CandidateID)

	require.Len(t, res.Exclusions, 2)
	byCandidate := map[kernel.ID]matchingdom.Exclusion{}
	for _, e := range res.Exclusions {
		byCandidate[e.CandidateID] = e
		assert.NotEmpty(t, e.Reason)
	}
	assert.Equal(t, matchingdom.GateLocation, byCandidate[cLagos].Gate)
	assert.Equal(t, matchingdom.GateMustHave, byCandidate[cWeak].Gate)
}

func TestGenerateShortlistRoleNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	_, err := d.shortlister().GenerateShortlist(context.Background(), kernel.NewID(), 10)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestGenerateShortlistBadScoreJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := validRole(t)
	c1 := kernel.NewID()
	d.roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{c1}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), c1).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), c1).Return(profileFor(t, c1), nil)
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: "not json"}, nil).Times(app.DefaultLLMAttempts)

	_, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func roleWithBand(t *testing.T, band kernel.SalaryBand) *role.Role {
	t.Helper()
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid,
			Responsibilities: []string{"build services"}, MustHaves: []string{"Go"}, SalaryBand: band,
		},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	return rl
}

// TestGenerateShortlistSkipsMissingData proves skipIfMissing: a recalled id with
// no candidate row, and one with no profile row, are both silently dropped
// (never scored, never excluded, no error).
func TestGenerateShortlistSkipsMissingData(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := validRole(t)
	noCand, noProfile := kernel.NewID(), kernel.NewID()

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{noCand, noProfile}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), noCand).Return(nil, kernel.NotFound("no candidate"))
	d.candidates.EXPECT().ByID(gomock.Any(), noProfile).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), noProfile).Return(nil, kernel.NotFound("no profile"))
	// No scoring, no persistence for either candidate.

	res, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, res.Matches)
	assert.Empty(t, res.Exclusions)
}

// TestGenerateShortlistSalaryGate proves the salary gate runs pre-scoring: an
// over-band candidate is excluded with a reason and never reaches the scorer.
func TestGenerateShortlistSalaryGate(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := roleWithBand(t, kernel.SalaryBand{Currency: "GHS", Low: 1000, High: 5000})
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra",
		talent.CandidateIntake{SalaryFloor: 8000, SalaryCurrency: "GHS"})
	require.NoError(t, err)

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{cid}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	// profiles.ByCandidateID and scorer.Complete are NEVER expected -> proves skip.

	res, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, res.Matches)
	require.Len(t, res.Exclusions, 1)
	assert.Equal(t, matchingdom.GateSalaryFloor, res.Exclusions[0].Gate)
	assert.Equal(t, cid, res.Exclusions[0].CandidateID)
}

// TestGenerateShortlistRejectsProtectedRubric proves the bias gate precedes all
// model and data access: a rubric naming a protected attribute is rejected
// before any embedding, recall, or scoring happens.
func TestGenerateShortlistRejectsProtectedRubric(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "age", Weight: 1.0, MustHave: true}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	// embedder.Embed / recaller.Recall / scorer.Complete are NEVER expected.

	_, err = d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

// TestGenerateShortlistRemoteRoleSkipsLocation proves a remote role (declared
// by a "remote" token in its location) disables the location gate: an
// out-of-city candidate survives.
func TestGenerateShortlistRemoteRoleSkipsLocation(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title: "Backend Engineer", Location: "Accra / Remote", Seniority: role.SeniorityMid,
			MustHaves: []string{"Go"},
		},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	cid := kernel.NewID()

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{cid}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidateAt(t, "Lagos"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profileFor(t, cid), nil)
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score09}, nil)
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	res, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)
	require.Len(t, res.Matches, 1)
	assert.Equal(t, cid, res.Matches[0].CandidateID)
	assert.Empty(t, res.Exclusions)
}

// TestGenerateShortlistPoolDepthExceedsPage proves the CAL-055 fix: pool_depth is
// the total strong-match count, not the page length. Three candidates all match,
// but a page size of 2 returns only the top two — while PoolDepth still reports 3.
func TestGenerateShortlistPoolDepthExceedsPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := validRole(t)
	c1, c2, c3 := kernel.NewID(), kernel.NewID(), kernel.NewID()

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{c1, c2, c3}, nil)
	for _, c := range []kernel.ID{c1, c2, c3} {
		d.candidates.EXPECT().ByID(gomock.Any(), c).Return(candidateAt(t, "Accra"), nil)
		d.profiles.EXPECT().ByCandidateID(gomock.Any(), c).Return(profileFor(t, c), nil)
	}
	// All three clear the must-have gate and are scored; all three are persisted.
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score09}, nil)
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score06}, nil)
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score06}, nil)
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(3)

	res, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 2)
	require.NoError(t, err)
	assert.Len(t, res.Matches, 2, "the page returns only the top two")
	assert.Equal(t, 3, res.PoolDepth, "but the pool depth reflects all three strong matches")
}

// TestCountAvailable proves the cheap, no-LLM instant signal (CAL-055/037):
// candidates that are logistically compatible AND whose verified profile covers
// the role's must-haves are counted; a location mismatch and a missing must-have
// are both excluded — and the scorer is never called.
func TestCountAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := validRole(t) // must-have: Go; location: Accra
	cGood, cLagos, cNoGo := kernel.NewID(), kernel.NewID(), kernel.NewID()

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]kernel.ID{cGood, cLagos, cNoGo}, nil)

	d.candidates.EXPECT().ByID(gomock.Any(), cGood).Return(candidateAt(t, "Accra"), nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cLagos).Return(candidateAt(t, "Lagos"), nil) // gated pre-profile
	d.candidates.EXPECT().ByID(gomock.Any(), cNoGo).Return(candidateAt(t, "Accra"), nil)

	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cGood).Return(profileFor(t, cGood), nil) // has Go
	// cLagos never has its profile loaded (logistically gated first).
	noGo, perr := talent.NewTalentProfile(cNoGo, "summary",
		[]talent.ProfileCompetency{{Name: "Python", Level: 5, EvidenceQuote: "built ETL"}})
	require.NoError(t, perr)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cNoGo).Return(noGo, nil)

	// No scorer.Complete expectation: CountAvailable must not invoke the LLM.
	n, err := d.shortlister().CountAvailable(context.Background(), rl.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only the Accra candidate covering the Go must-have counts")
}
