package matching_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

func TestRefinerRevisesPersistsAndReRanks(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := validRole(t)
	cid := kernel.NewID()

	// Refiner loads + persists the revised role; GenerateShortlist loads it again.
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil).Times(2)
	d.roles.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{cid}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profileFor(t, cid), nil)
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: score09}, nil)
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	refiner := matchingapp.NewRefiner(d.roles, d.shortlister())
	newRubric := role.Rubric{Competencies: []role.Competency{
		{Name: "Go", Weight: 3, MustHave: true}, {Name: "SQL", Weight: 1},
	}}
	res, err := refiner.Refine(context.Background(), rl.ID, rl.EmployerID, rl.Spec, newRubric, 10)
	require.NoError(t, err)
	require.Len(t, res.Matches, 1)
	assert.Equal(t, cid, res.Matches[0].CandidateID)
	// the override rubric was re-normalized in place before ranking
	assert.InDelta(t, 0.75, rl.Rubric.Competencies[0].Weight, 0.01)
}

func TestRefinerRoleNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	refiner := matchingapp.NewRefiner(d.roles, d.shortlister())
	_, err := refiner.Refine(context.Background(), kernel.NewID(), kernel.NewID(), role.RoleSpec{}, role.Rubric{}, 10)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

// TestRefinerRejectsOtherEmployer locks the Flow A IDOR guard (CAL-116): an
// employer may refine their OWN role only. A non-owner is forbidden and the role
// is never revised or persisted (no Update, no re-rank).
func TestRefinerRejectsOtherEmployer(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := validRole(t)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	refiner := matchingapp.NewRefiner(d.roles, d.shortlister())
	_, err := refiner.Refine(context.Background(), rl.ID, kernel.NewID(), rl.Spec, rl.Rubric, 10)
	assert.Equal(t, kernel.KindForbidden, kernel.KindOf(err))
}
