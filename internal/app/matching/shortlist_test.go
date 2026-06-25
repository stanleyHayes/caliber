package matching_test

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	score06 = `{"overall_score":0.6,"confidence":"medium","breakdown":[{"competency":"Go","score":3,"evidence":"x"}],"rationale":"solid","watch_outs":[],"thin_evidence":false}`
	score09 = `{"overall_score":0.9,"confidence":"high","breakdown":[{"competency":"Go","score":4.5,"evidence":"y"}],"rationale":"excellent","watch_outs":["thin mentoring"],"thin_evidence":false}`
)

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

func profileFor(t *testing.T, cid kernel.ID) *talent.TalentProfile {
	t.Helper()
	p, err := talent.NewTalentProfile(cid, "summary",
		[]talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "built services"}})
	require.NoError(t, err)
	return p
}

func TestGenerateShortlistRanksAndPersists(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	recaller := mocks.NewMockCandidateRecaller(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	scorer := mocks.NewMockLLMClient(ctrl)
	matchRepo := mocks.NewMockMatchRepository(ctrl)

	rl := validRole(t)
	c1, c2 := kernel.NewID(), kernel.NewID()

	roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1, 0.2}, nil)
	recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), 10).Return([]kernel.ID{c1, c2}, nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), c1).Return(profileFor(t, c1), nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), c2).Return(profileFor(t, c2), nil)
	gomock.InOrder(
		scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score06}, nil),
		scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score09}, nil),
	)
	matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	s := matchingapp.NewShortlister(roles, profiles, recaller, embedder, scorer, matchRepo)
	out, err := s.GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, c2, out[0].CandidateID, "candidate scored 0.9 ranks first")
	assert.InDelta(t, 0.9, out[0].OverallScore, 1e-9)
	assert.Equal(t, kernel.ConfidenceHigh, out[0].Confidence)
	assert.Equal(t, c1, out[1].CandidateID)
	assert.InDelta(t, 0.6, out[1].OverallScore, 1e-9)
}

func TestGenerateShortlistRoleNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	s := matchingapp.NewShortlister(roles, mocks.NewMockTalentProfileRepository(ctrl), mocks.NewMockCandidateRecaller(ctrl),
		mocks.NewMockEmbedder(ctrl), mocks.NewMockLLMClient(ctrl), mocks.NewMockMatchRepository(ctrl))
	_, err := s.GenerateShortlist(context.Background(), kernel.NewID(), 10)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestGenerateShortlistBadScoreJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	recaller := mocks.NewMockCandidateRecaller(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	scorer := mocks.NewMockLLMClient(ctrl)

	rl := validRole(t)
	c1 := kernel.NewID()
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{c1}, nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), c1).Return(profileFor(t, c1), nil)
	scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: "not json"}, nil)

	s := matchingapp.NewShortlister(roles, profiles, recaller, embedder, scorer, mocks.NewMockMatchRepository(ctrl))
	_, err := s.GenerateShortlist(context.Background(), rl.ID, 10)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}
