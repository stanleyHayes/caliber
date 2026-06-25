package grpcadapter

import (
	"errors"
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrToStatus(t *testing.T) {
	cases := map[kernel.Kind]codes.Code{
		kernel.KindInvalid:      codes.InvalidArgument,
		kernel.KindNotFound:     codes.NotFound,
		kernel.KindConflict:     codes.AlreadyExists,
		kernel.KindUnauthorized: codes.Unauthenticated,
		kernel.KindForbidden:    codes.PermissionDenied,
		kernel.KindInternal:     codes.Internal,
	}
	for kind, want := range cases {
		assert.Equal(t, want, status.Code(errToStatus(&kernel.Error{Kind: kind, Msg: "x"})))
	}
	assert.Equal(t, codes.Internal, status.Code(errToStatus(errors.New("plain"))))
}

func TestEnumMappings(t *testing.T) {
	assert.Equal(t, caliberv1.Seniority_SENIORITY_LEAD, seniorityToProto(role.SeniorityLead))
	assert.Equal(t, caliberv1.Seniority_SENIORITY_UNSPECIFIED, seniorityToProto(role.Seniority(99)))
	assert.Equal(t, caliberv1.RoleStatus_ROLE_STATUS_CLOSED, roleStatusToProto(role.RoleClosed))
	assert.Equal(t, caliberv1.RoleStatus_ROLE_STATUS_UNSPECIFIED, roleStatusToProto(role.RoleStatus(99)))
}
