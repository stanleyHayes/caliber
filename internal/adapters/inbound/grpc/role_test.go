package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newServer() *RoleServer {
	return NewRoleServer(roles.NewSpecGenerator(llm.NewDev(), memory.NewRoleRepo(), time.Now))
}

func TestGenerateRoleSpecHandler(t *testing.T) {
	resp, err := newServer().GenerateRoleSpec(context.Background(),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: kernel.NewID().String(), FreeText: "Senior Go engineer in Accra"})
	if err != nil {
		t.Fatalf("GenerateRoleSpec: %v", err)
	}
	got := resp.GetRole()
	if got.GetTitle() == "" || got.GetStatus() != caliberv1.RoleStatus_ROLE_STATUS_DRAFT {
		t.Errorf("unexpected role: %+v", got)
	}
	if len(got.GetRubric().GetCompetencies()) == 0 {
		t.Error("expected rubric competencies")
	}
	if got.GetSpec().GetSalaryBand() == nil {
		t.Error("expected salary band")
	}
}

func TestGenerateRoleSpecInvalid(t *testing.T) {
	_, err := newServer().GenerateRoleSpec(context.Background(),
		&caliberv1.GenerateRoleSpecRequest{EmployerId: "", FreeText: "x"})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestEnumMappings(t *testing.T) {
	if seniorityToProto(role.SeniorityLead) != caliberv1.Seniority_SENIORITY_LEAD {
		t.Error("seniority lead map")
	}
	if seniorityToProto(role.Seniority(99)) != caliberv1.Seniority_SENIORITY_UNSPECIFIED {
		t.Error("unknown seniority -> unspecified")
	}
	if roleStatusToProto(role.RoleClosed) != caliberv1.RoleStatus_ROLE_STATUS_CLOSED {
		t.Error("status closed map")
	}
	if roleStatusToProto(role.RoleStatus(99)) != caliberv1.RoleStatus_ROLE_STATUS_UNSPECIFIED {
		t.Error("unknown status -> unspecified")
	}
}
