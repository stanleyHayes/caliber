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
	return NewRoleServer(roles.NewSpecGenerator(llm.NewDev(), memory.NewRoleRepo(), time.Now))
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
