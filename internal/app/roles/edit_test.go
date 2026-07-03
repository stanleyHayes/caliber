package roles_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/mocks"
)

func sampleRole(t *testing.T) *role.Role {
	t.Helper()
	r, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	return r
}

func TestSpecEditorUpdateReweights(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRoleRepository(ctrl)
	r := sampleRole(t)
	repo.EXPECT().ByID(gomock.Any(), r.ID).Return(r, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	editor := roles.NewSpecEditor(repo)
	out, err := editor.Update(context.Background(), r.ID, r.Spec,
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 3, MustHave: true}, {Name: "SQL", Weight: 1}}})
	require.NoError(t, err)

	require.Len(t, out.Rubric.Competencies, 2)
	assert.InDelta(t, 0.75, out.Rubric.Competencies[0].Weight, 0.01, "re-weighting normalized to sum 1.0")
	assert.InDelta(t, 0.25, out.Rubric.Competencies[1].Weight, 0.01)
}

func TestSpecEditorUpdateRefreshesEmbedding(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRoleRepository(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	r := sampleRole(t)
	repo.EXPECT().ByID(gomock.Any(), r.ID).Return(r, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, text string) ([]float32, error) {
		assert.Contains(t, text, "Staff Engineer")
		return []float32{0.3, 0.9}, nil
	})
	var updated *role.Role
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r *role.Role) error {
		updated = r
		return nil
	})

	out, err := roles.NewSpecEditor(repo, roles.WithEditorEmbedder(embedder)).
		Update(context.Background(), r.ID,
			role.RoleSpec{Title: "Staff Engineer", Location: "Accra", Seniority: role.SenioritySenior},
			r.Rubric)
	require.NoError(t, err)
	assert.Equal(t, []float32{0.3, 0.9}, out.Embedding)
	require.NotNil(t, updated)
	assert.Equal(t, []float32{0.3, 0.9}, updated.Embedding)
}

func TestSpecEditorEmbeddingErrorStopsBeforeUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRoleRepository(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	r := sampleRole(t)
	repo.EXPECT().ByID(gomock.Any(), r.ID).Return(r, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return(nil, errors.New("embed down"))

	_, err := roles.NewSpecEditor(repo, roles.WithEditorEmbedder(embedder)).
		Update(context.Background(), r.ID, r.Spec, r.Rubric)
	require.Error(t, err)
}

func TestSpecEditorUpdateRejectsInvalidSpec(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRoleRepository(ctrl)
	r := sampleRole(t)
	repo.EXPECT().ByID(gomock.Any(), r.ID).Return(r, nil)
	// Update must NOT be called: validation fails first.

	editor := roles.NewSpecEditor(repo)
	_, err := editor.Update(context.Background(), r.ID,
		role.RoleSpec{Title: "", Seniority: role.SeniorityMid}, // empty title -> invalid
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1}}})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestSpecEditorGetNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRoleRepository(ctrl)
	id := kernel.NewID()
	repo.EXPECT().ByID(gomock.Any(), id).Return(nil, kernel.NotFound("nope"))
	_, err := roles.NewSpecEditor(repo).Get(context.Background(), id)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestSpecEditorRequiresID(t *testing.T) {
	editor := roles.NewSpecEditor(mocks.NewMockRoleRepository(gomock.NewController(t)))
	_, gErr := editor.Get(context.Background(), kernel.ID(""))
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(gErr))
	_, uErr := editor.Update(context.Background(), kernel.ID(""), role.RoleSpec{}, role.Rubric{})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(uErr))
}
