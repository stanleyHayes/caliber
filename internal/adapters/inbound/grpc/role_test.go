package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newServer() *RoleServer {
	repo := memory.NewRoleRepo()
	return NewRoleServer(roles.NewSpecGenerator(llm.NewDev(), repo, time.Now), roles.NewSpecEditor(repo), nil)
}

// generatedRole creates a role owned by emp (the authenticated employer).
func generatedRole(t *testing.T, srv *RoleServer, emp kernel.ID) *caliberv1.Role {
	t.Helper()
	gen, err := srv.GenerateRoleSpec(asEmployer(context.Background(), emp),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: "Senior Go engineer in Accra"})
	require.NoError(t, err)
	return gen.GetRole()
}

func TestGenerateRoleSpecHandler(t *testing.T) {
	emp := kernel.NewID()
	resp, err := newServer().GenerateRoleSpec(asEmployer(context.Background(), emp),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: "Senior Go engineer in Accra"})
	require.NoError(t, err)
	got := resp.GetRole()
	assert.NotEmpty(t, got.GetTitle())
	assert.Equal(t, caliberv1.RoleStatus_ROLE_STATUS_DRAFT, got.GetStatus())
	assert.NotEmpty(t, got.GetRubric().GetCompetencies())
	assert.NotNil(t, got.GetSpec().GetSalaryBand())
}

type stubCounter struct{ n int }

func (c stubCounter) CountAvailable(context.Context, kernel.ID) (int, error) { return c.n, nil }

func TestGenerateRoleSpecSurfacesAvailableMatches(t *testing.T) {
	repo := memory.NewRoleRepo()
	srv := NewRoleServer(roles.NewSpecGenerator(llm.NewDev(), repo, time.Now), roles.NewSpecEditor(repo), stubCounter{n: 7})
	emp := kernel.NewID()
	resp, err := srv.GenerateRoleSpec(asEmployer(context.Background(), emp),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: "Senior Go engineer in Accra"})
	require.NoError(t, err)
	assert.Equal(t, int32(7), resp.GetAvailableMatches(), "the instant pool-depth signal is returned with the role")
}

func TestGenerateRoleSpecInvalid(t *testing.T) {
	// Authenticated as the employer, but an empty hiring-need text is rejected.
	emp := kernel.NewID()
	_, err := newServer().GenerateRoleSpec(asEmployer(context.Background(), emp),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: "   "})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetRoleHandler(t *testing.T) {
	srv := newServer()
	emp := kernel.NewID()
	id := generatedRole(t, srv, emp).GetId()
	got, err := srv.GetRole(asEmployer(context.Background(), emp), &caliberv1.GetRoleRequest{RoleId: id})
	require.NoError(t, err)
	assert.Equal(t, id, got.GetRole().GetId())
}

func TestGetRoleNotFound(t *testing.T) {
	_, err := newServer().GetRole(asEmployer(context.Background(), kernel.NewID()),
		&caliberv1.GetRoleRequest{RoleId: kernel.NewID().String()})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateRoleSpecReweights(t *testing.T) {
	srv := newServer()
	emp := kernel.NewID()
	role := generatedRole(t, srv, emp)
	resp, err := srv.UpdateRoleSpec(asEmployer(context.Background(), emp), &caliberv1.UpdateRoleSpecRequest{
		RoleId: role.GetId(),
		Spec:   role.GetSpec(),
		Rubric: &caliberv1.Rubric{Competencies: []*caliberv1.Competency{
			{Name: "Go", Weight: 3, MustHave: true},
			{Name: "SQL", Weight: 1},
		}},
	})
	require.NoError(t, err)
	comps := resp.GetRole().GetRubric().GetCompetencies()
	require.Len(t, comps, 2)
	var sum float64
	for _, c := range comps {
		sum += c.GetWeight()
	}
	assert.InDelta(t, 1.0, sum, 0.01, "weights re-normalized to 1.0")
	assert.InDelta(t, 0.75, comps[0].GetWeight(), 0.01)
}

func TestUpdateRoleSpecNotFound(t *testing.T) {
	srv := newServer()
	_, err := srv.UpdateRoleSpec(asEmployer(context.Background(), kernel.NewID()), &caliberv1.UpdateRoleSpecRequest{
		RoleId: kernel.NewID().String(),
		Spec:   &caliberv1.RoleSpec{Title: "X", Seniority: caliberv1.Seniority_SENIORITY_MID},
		Rubric: &caliberv1.Rubric{Competencies: []*caliberv1.Competency{{Name: "Go", Weight: 1}}},
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateRoleSpecRejectsOtherEmployer(t *testing.T) {
	srv := newServer()
	emp := kernel.NewID()
	role := generatedRole(t, srv, emp)
	// A different employer cannot edit emp's role (IDOR).
	_, err := srv.UpdateRoleSpec(asEmployer(context.Background(), kernel.NewID()), &caliberv1.UpdateRoleSpecRequest{
		RoleId: role.GetId(),
		Spec:   role.GetSpec(),
		Rubric: &caliberv1.Rubric{Competencies: []*caliberv1.Competency{{Name: "Go", Weight: 1, MustHave: true}}},
	})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestListRolesHandler(t *testing.T) {
	srv := newServer()
	emp := kernel.NewID()
	for _, txt := range []string{"Go engineer Accra", "Frontend engineer Kumasi"} {
		_, err := srv.GenerateRoleSpec(asEmployer(context.Background(), emp),
			&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: txt})
		require.NoError(t, err)
	}
	resp, err := srv.ListRoles(asEmployer(context.Background(), emp),
		&caliberv1.ListRolesRequest{EmployerId: emp.String(), Page: &caliberv1.PageRequest{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	assert.Len(t, resp.GetRoles(), 2)
	assert.Equal(t, int64(2), resp.GetPage().GetTotalItems())
}

func TestRoleWritesRequireReviewer(t *testing.T) {
	srv := newServer()
	emp := kernel.NewID()
	// A candidate cannot create a role.
	_, err := srv.GenerateRoleSpec(asCandidate(context.Background(), emp),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: "Senior Go engineer"})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	// An unauthenticated caller cannot create a role.
	_, err = srv.GenerateRoleSpec(context.Background(),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: "Senior Go engineer"})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	// An employer cannot create a role under ANOTHER employer's id (IDOR).
	_, err = srv.GenerateRoleSpec(asEmployer(context.Background(), kernel.NewID()),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: "Senior Go engineer"})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}
