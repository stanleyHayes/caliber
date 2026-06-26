package candidateagent_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	agentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"
)

const assessJSON = `{"fit_score":0.85,"apply":true,"tailored_summary":"Drawing on verified Go experience, a strong fit."}`

type deps struct {
	candidates *mocks.MockCandidateRepository
	profiles   *mocks.MockTalentProfileRepository
	roles      *mocks.MockRoleRepository
	apps       *mocks.MockApplicationRepository
	llm        *mocks.MockLLMClient
}

func newDeps(ctrl *gomock.Controller) deps {
	return deps{
		candidates: mocks.NewMockCandidateRepository(ctrl),
		profiles:   mocks.NewMockTalentProfileRepository(ctrl),
		roles:      mocks.NewMockRoleRepository(ctrl),
		apps:       mocks.NewMockApplicationRepository(ctrl),
		llm:        mocks.NewMockLLMClient(ctrl),
	}
}

func (d deps) runner() *agentapp.AgentRunner {
	return agentapp.NewAgentRunner(d.candidates, d.profiles, d.roles, d.apps, d.llm)
}

func candidate(t *testing.T, location string) *talent.Candidate {
	t.Helper()
	c, err := talent.NewCandidate(kernel.NewID(), location, talent.CandidateIntake{})
	require.NoError(t, err)
	return c
}

func profileWith(t *testing.T, cid kernel.ID, comps ...talent.ProfileCompetency) *talent.TalentProfile {
	t.Helper()
	p, err := talent.NewTalentProfile(cid, "summary", comps)
	require.NoError(t, err)
	return p
}

func openRole(t *testing.T, comps []role.Competency) *role.Role {
	t.Helper()
	r, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: comps}, time.Unix(1, 0))
	require.NoError(t, err)
	return r
}

func TestRunSubmitsHonestStrongMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	cand := candidate(t, "Accra")
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "built services"})
	rl := openRole(t, []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: assessJSON}, nil)

	var created *agentdom.Application
	d.apps.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, a *agentdom.Application) error { created = a; return nil })

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, view.NewMatches)
	assert.Equal(t, 1, view.ApplicationsSubmitted)
	require.NotNil(t, created)
	assert.Equal(t, profile.ID, created.ProfileID, "grounded in the verified profile (no fabrication)")
	assert.Equal(t, agentdom.SourceAgent, created.Source)
	assert.Equal(t, agentdom.StatusSubmitted, created.Status)
}

func TestRunSkipsRoleWithUncoveredMustHave(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	// Profile has Go but the role must-have is Rust -> applying would require fabrication.
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "x"})
	rl := openRole(t, []role.Competency{{Name: "Rust", Weight: 1, MustHave: true}})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidate(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	// llm.Complete and apps.Create must NOT be called.

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Zero(t, view.NewMatches)
	assert.Zero(t, view.ApplicationsSubmitted)
}

func TestRunSkipsLocationMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "x"})
	rl := openRole(t, []role.Competency{{Name: "Go", Weight: 1, MustHave: true}})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidate(t, "Lagos"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Zero(t, view.NewMatches)
}

func TestRunRespectsAgentDecisionNotToApply(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "x"})
	rl := openRole(t, []role.Competency{{Name: "Go", Weight: 1, MustHave: true}})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidate(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(
		app.LLMResponse{Text: `{"fit_score":0.3,"apply":false,"tailored_summary":""}`}, nil)
	// apps.Create must NOT be called.

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, view.NewMatches, "eligible and considered")
	assert.Zero(t, view.ApplicationsSubmitted, "but the agent chose not to apply")
}

func TestRunSkipsBlankSummary(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "x"})
	rl := openRole(t, []role.Competency{{Name: "Go", Weight: 1, MustHave: true}})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidate(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	// A blank summary would be ungrounded -> NewAgentApplication rejects it -> no application.
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(
		app.LLMResponse{Text: `{"fit_score":0.9,"apply":true,"tailored_summary":"   "}`}, nil)

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Zero(t, view.ApplicationsSubmitted)
}

func TestRunNoProfileDoesNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidate(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("no profile"))
	// roles.ListOpen must NOT be called: without a verified profile the agent cannot act.

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Zero(t, view.ApplicationsSubmitted)
}
