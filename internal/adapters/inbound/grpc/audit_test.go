package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

func TestAuditServer_ListAuditLog(t *testing.T) {
	repo := memory.NewAuditRepo()
	actor, entityID := kernel.NewID(), kernel.NewID()
	e1, _ := audit.NewAuditEntry(actor, audit.ActionContestRaised, "contest", entityID, "", "", time.Unix(1, 0))
	e2, _ := audit.NewAuditEntry(actor, audit.ActionContestResolved, "contest", entityID, "", "", time.Unix(2, 0))
	require.NoError(t, repo.Append(context.Background(), e1))
	require.NoError(t, repo.Append(context.Background(), e2))

	srv := NewAuditServer(repo)
	empCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleEmployer.String()})

	resp, err := srv.ListAuditLog(empCtx, &caliberv1.ListAuditLogRequest{
		Entity: "contest", EntityId: entityID.String(),
		Page: &caliberv1.PageRequest{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetEntries(), 2)
	assert.Equal(t, audit.ActionContestResolved, resp.GetEntries()[0].GetAction(), "newest first")
	assert.Equal(t, int64(2), resp.GetPage().GetTotalItems())
}

func TestAuditServer_AuthzAndValidation(t *testing.T) {
	srv := NewAuditServer(memory.NewAuditRepo())
	req := &caliberv1.ListAuditLogRequest{Entity: "contest", EntityId: kernel.NewID().String()}

	// candidate may not read the audit trail
	candCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleCandidate.String()})
	_, err := srv.ListAuditLog(candCtx, req)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// unauthenticated -> unauthenticated
	_, err = srv.ListAuditLog(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	// employer with missing filters -> invalid argument
	empCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleEmployer.String()})
	_, err = srv.ListAuditLog(empCtx, &caliberv1.ListAuditLogRequest{Entity: "contest"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
