package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const matchScoreJSON = `{"overall_score":0.8,"confidence":"high","breakdown":[{"competency":"Go","score":4,"evidence":"built services"}],"rationale":"strong","watch_outs":["mentoring"],"thin_evidence":false}`

func shortlisterWithOneMatch(t *testing.T, ctrl *gomock.Controller, rl *role.Role, cid kernel.ID) *matchingapp.Shortlister {
	t.Helper()
	roles := mocks.NewMockRoleRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	recaller := mocks.NewMockCandidateRecaller(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	scorer := mocks.NewMockLLMClient(ctrl)
	matchRepo := mocks.NewMockMatchRepository(ctrl)

	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{cid}, nil)
	prof, err := talent.NewTalentProfile(cid, "s", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(prof, nil)
	scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: matchScoreJSON}, nil)
	matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	return matchingapp.NewShortlister(roles, profiles, recaller, embedder, scorer, matchRepo)
}

func TestGenerateShortlistHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1, MustHave: true}}},
		time.Unix(1, 0))
	require.NoError(t, err)
	cid := kernel.NewID()

	srv := NewMatchServer(shortlisterWithOneMatch(t, ctrl, rl, cid))
	resp, err := srv.GenerateShortlist(context.Background(),
		&caliberv1.GenerateShortlistRequest{RoleId: rl.ID.String(), Page: &caliberv1.PageRequest{PageSize: 5}})
	require.NoError(t, err)

	matches := resp.GetShortlist().GetMatches()
	require.Len(t, matches, 1)
	m := matches[0]
	assert.Equal(t, cid.String(), m.GetCandidateId())
	assert.InDelta(t, 0.8, m.GetOverallScore(), 1e-9)
	assert.Equal(t, caliberv1.Confidence_CONFIDENCE_HIGH, m.GetConfidence())
	require.Len(t, m.GetBreakdown(), 1)
	assert.Equal(t, "Go", m.GetBreakdown()[0].GetCompetency())
	assert.Equal(t, []string{"mentoring"}, m.GetWatchOuts())
	assert.Equal(t, int32(1), resp.GetShortlist().GetPoolDepth())
}

func TestGenerateShortlistHandlerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	s := matchingapp.NewShortlister(roles, mocks.NewMockTalentProfileRepository(ctrl), mocks.NewMockCandidateRecaller(ctrl),
		mocks.NewMockEmbedder(ctrl), mocks.NewMockLLMClient(ctrl), mocks.NewMockMatchRepository(ctrl))
	_, err := NewMatchServer(s).GenerateShortlist(context.Background(), &caliberv1.GenerateShortlistRequest{RoleId: "x"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}
