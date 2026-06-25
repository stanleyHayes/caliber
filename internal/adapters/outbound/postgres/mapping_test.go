package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

func TestStatusMapping(t *testing.T) {
	for _, s := range []role.RoleStatus{role.RoleDraft, role.RoleOpen, role.RoleClosed} {
		assert.Equal(t, s, statusFromDB(statusToDB(s)), "round-trip status %d", s)
	}
	assert.Equal(t, "draft", statusToDB(role.RoleStatus(99)))
	assert.Equal(t, role.RoleDraft, statusFromDB("nonsense"))
}

func TestRoleRoundTrip(t *testing.T) {
	emp := kernel.NewID()
	rl, err := role.NewRole(emp,
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SenioritySenior,
			SalaryBand: kernel.SalaryBand{Currency: "GHS", Low: 1000, High: 2000}},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	_ = rl.Open()

	spec, rubric, _, err := marshalRole(rl)
	require.NoError(t, err)

	got, err := toDomainRole(rl.ID.String(), emp.String(), rl.Title, statusToDB(rl.Status), spec, rubric,
		pgtype.Timestamptz{Time: rl.CreatedAt, Valid: true})
	require.NoError(t, err)
	assert.Equal(t, rl.ID, got.ID)
	assert.Equal(t, emp, got.EmployerID)
	assert.Equal(t, "Backend Engineer", got.Title)
	assert.Equal(t, role.RoleOpen, got.Status)
	assert.Equal(t, role.SenioritySenior, got.Spec.Seniority)
	assert.InDelta(t, 1.0, got.Rubric.TotalWeight(), 0.001)
	assert.True(t, got.CreatedAt.Equal(rl.CreatedAt))
}

func TestClampInt32(t *testing.T) {
	assert.Equal(t, int32(0), clampInt32(-5))
	assert.Equal(t, int32(20), clampInt32(20))
}

func TestIsUniqueViolation(t *testing.T) {
	assert.False(t, isUniqueViolation(nil))
	assert.False(t, isUniqueViolation(assertAnError{}))
}

type assertAnError struct{}

func (assertAnError) Error() string { return "x" }
