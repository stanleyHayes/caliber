package memory

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkRole(t *testing.T, emp kernel.ID, title string, ts time.Time) *role.Role {
	t.Helper()
	r, err := role.NewRole(emp,
		role.RoleSpec{Title: title, Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Core", Weight: 1, MustHave: true}}},
		ts)
	require.NoError(t, err)
	return r
}

func TestRoleRepoCRUD(t *testing.T) {
	ctx := context.Background()
	repo := NewRoleRepo()
	emp := kernel.NewID()
	r := mkRole(t, emp, "Engineer", time.Unix(1000, 0))

	require.NoError(t, repo.Create(ctx, r))
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(repo.Create(ctx, r)))

	got, err := repo.ByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "Engineer", got.Title)

	_, err = repo.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))

	r.Title = "Senior Engineer"
	require.NoError(t, repo.Update(ctx, r))
	got, err = repo.ByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "Senior Engineer", got.Title)

	ghost := mkRole(t, emp, "ghost", time.Unix(1, 0))
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(repo.Update(ctx, ghost)))
}

func TestRoleRepoListPagination(t *testing.T) {
	ctx := context.Background()
	repo := NewRoleRepo()
	empA, empB := kernel.NewID(), kernel.NewID()
	for i := range 3 {
		require.NoError(t, repo.Create(ctx, mkRole(t, empA, "A", time.Unix(int64(i+1), 0))))
	}
	require.NoError(t, repo.Create(ctx, mkRole(t, empB, "B", time.Unix(99, 0))))

	page1, total, err := repo.ListByEmployer(ctx, empA, kernel.NewPage(1, 2))
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, page1, 2)

	page2, _, err := repo.ListByEmployer(ctx, empA, kernel.NewPage(2, 2))
	require.NoError(t, err)
	assert.Len(t, page2, 1)
	assert.True(t, page1[0].CreatedAt.After(page1[1].CreatedAt), "results should be newest-first")
}
