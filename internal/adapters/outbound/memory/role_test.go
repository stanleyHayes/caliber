package memory

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

func mkRole(t *testing.T, emp kernel.ID, title string, ts time.Time) *role.Role {
	t.Helper()
	r, err := role.NewRole(emp,
		role.RoleSpec{Title: title, Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Core", Weight: 1, MustHave: true}}},
		ts)
	if err != nil {
		t.Fatalf("NewRole: %v", err)
	}
	return r
}

func TestRoleRepoCRUD(t *testing.T) {
	ctx := context.Background()
	repo := NewRoleRepo()
	emp := kernel.NewID()
	r := mkRole(t, emp, "Engineer", time.Unix(1000, 0))

	if err := repo.Create(ctx, r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Create(ctx, r); kernel.KindOf(err) != kernel.KindConflict {
		t.Error("duplicate Create should conflict")
	}
	got, err := repo.ByID(ctx, r.ID)
	if err != nil || got.Title != "Engineer" {
		t.Errorf("ByID: %v", err)
	}
	if _, err := repo.ByID(ctx, kernel.NewID()); kernel.KindOf(err) != kernel.KindNotFound {
		t.Error("missing ByID should be not found")
	}

	r.Title = "Senior Engineer"
	if err := repo.Update(ctx, r); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = repo.ByID(ctx, r.ID)
	if got.Title != "Senior Engineer" {
		t.Error("Update did not persist")
	}
	if err := repo.Update(ctx, mkRole(t, emp, "ghost", time.Unix(1, 0))); kernel.KindOf(err) != kernel.KindNotFound {
		t.Error("Update missing should be not found")
	}
}

func TestRoleRepoListPagination(t *testing.T) {
	ctx := context.Background()
	repo := NewRoleRepo()
	empA, empB := kernel.NewID(), kernel.NewID()
	for i := range 3 {
		_ = repo.Create(ctx, mkRole(t, empA, "A", time.Unix(int64(i+1), 0)))
	}
	_ = repo.Create(ctx, mkRole(t, empB, "B", time.Unix(99, 0)))

	page1, total, err := repo.ListByEmployer(ctx, empA, kernel.NewPage(1, 2))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(page1) != 2 {
		t.Errorf("page1: total=%d len=%d, want 3/2", total, len(page1))
	}
	page2, _, _ := repo.ListByEmployer(ctx, empA, kernel.NewPage(2, 2))
	if len(page2) != 1 {
		t.Errorf("page2 len=%d, want 1", len(page2))
	}
	// newest first
	if !page1[0].CreatedAt.After(page1[1].CreatedAt) {
		t.Error("results should be newest-first")
	}
}
