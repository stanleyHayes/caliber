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
	return NewRoleServer(roles.NewSpecGenerator(llm.NewDev(), repo, time.Now), roles.NewSpecEditor(repo))
}

func generatedRole(t *testing.T, srv *RoleServer) *caliberv1.Role {
	t.Helper()
	gen, err := srv.GenerateRoleSpec(context.Background(),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: kernel.NewID().String(), FreeText: "Senior Go engineer in Accra"})
	require.NoError(t, err)
	return gen.GetRole()
}

func TestGenerateRoleSpecHandler(t *testing.T) {
	resp, err := newServer().GenerateRoleSpec(context.Background(),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: kernel.NewID().String(), FreeText: "Senior Go engineer in Accra"})
	require.NoError(t, err)
	got := resp.GetRole()
	assert.NotEmpty(t, got.GetTitle())
	assert.Equal(t, caliberv1.RoleStatus_ROLE_STATUS_DRAFT, got.GetStatus())
	assert.NotEmpty(t, got.GetRubric().GetCompetencies())
	assert.NotNil(t, got.GetSpec().GetSalaryBand())
}

func TestGenerateRoleSpecInvalid(t *testing.T) {
	_, err := newServer().GenerateRoleSpec(context.Background(),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: "", FreeText: "x"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetRoleHandler(t *testing.T) {
	srv := newServer()
	id := generatedRole(t, srv).GetId()
	got, err := srv.GetRole(context.Background(), &caliberv1.GetRoleRequest{RoleId: id})
	require.NoError(t, err)
	assert.Equal(t, id, got.GetRole().GetId())
}

func TestGetRoleNotFound(t *testing.T) {
	_, err := newServer().GetRole(context.Background(), &caliberv1.GetRoleRequest{RoleId: kernel.NewID().String()})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateRoleSpecReweights(t *testing.T) {
	srv := newServer()
	role := generatedRole(t, srv)
	resp, err := srv.UpdateRoleSpec(context.Background(), &caliberv1.UpdateRoleSpecRequest{
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
	_, err := srv.UpdateRoleSpec(context.Background(), &caliberv1.UpdateRoleSpecRequest{
		RoleId: kernel.NewID().String(),
		Spec:   &caliberv1.RoleSpec{Title: "X", Seniority: caliberv1.Seniority_SENIORITY_MID},
		Rubric: &caliberv1.Rubric{Competencies: []*caliberv1.Competency{{Name: "Go", Weight: 1}}},
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestListRolesHandler(t *testing.T) {
	srv := newServer()
	emp := kernel.NewID()
	for _, txt := range []string{"Go engineer Accra", "Frontend engineer Kumasi"} {
		_, err := srv.GenerateRoleSpec(context.Background(),
			&caliberv1.GenerateRoleSpecRequest{EmployerId: emp.String(), FreeText: txt})
		require.NoError(t, err)
	}
	resp, err := srv.ListRoles(context.Background(),
		&caliberv1.ListRolesRequest{EmployerId: emp.String(), Page: &caliberv1.PageRequest{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	assert.Len(t, resp.GetRoles(), 2)
	assert.Equal(t, int64(2), resp.GetPage().GetTotalItems())
}
