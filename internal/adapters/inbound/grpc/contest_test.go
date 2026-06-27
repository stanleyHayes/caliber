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
	contestapp "github.com/xcreativs/caliber/internal/app/contest"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

func contestServer() *ContestServer {
	return NewContestServer(contestapp.NewService(memory.NewContestRepo(), memory.NewAuditRepo(), time.Now))
}

func asRole(ctx context.Context, role identity.Role) context.Context {
	return context.WithValue(ctx, principalKey{}, app.Principal{UserID: kernel.NewID(), Role: role.String()})
}

func TestContestServer_RaiseAndListAsCandidate(t *testing.T) {
	srv := contestServer()
	cid := kernel.NewID()
	ctx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: cid, Role: identity.RoleCandidate.String()})

	resp, err := srv.RaiseContest(ctx, &caliberv1.RaiseContestRequest{
		Subject:   caliberv1.ContestSubject_CONTEST_SUBJECT_MATCH,
		SubjectId: kernel.NewID().String(),
		Reason:    "the breakdown ignored my recent Go work",
	})
	require.NoError(t, err)
	assert.Equal(t, cid.String(), resp.GetContest().GetCandidateId(), "raised as the authenticated candidate, not a body field")
	assert.Equal(t, caliberv1.ContestStatus_CONTEST_STATUS_OPEN, resp.GetContest().GetStatus())
	assert.Equal(t, caliberv1.ContestSubject_CONTEST_SUBJECT_MATCH, resp.GetContest().GetSubject())

	list, err := srv.ListMyContests(ctx, &caliberv1.ListMyContestsRequest{Page: &caliberv1.PageRequest{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	require.Len(t, list.GetContests(), 1)
	assert.Equal(t, int64(1), list.GetPage().GetTotalItems())

	// listing is candidate-only: an employer is forbidden, not given an empty list
	_, err = srv.ListMyContests(asRole(context.Background(), identity.RoleEmployer),
		&caliberv1.ListMyContestsRequest{Page: &caliberv1.PageRequest{Page: 1, PageSize: 10}})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestContestServer_RaiseRejectsNonCandidate(t *testing.T) {
	srv := contestServer()
	req := &caliberv1.RaiseContestRequest{Subject: caliberv1.ContestSubject_CONTEST_SUBJECT_MATCH, SubjectId: kernel.NewID().String(), Reason: "r"}

	// authenticated employer -> permission denied
	_, err := srv.RaiseContest(asRole(context.Background(), identity.RoleEmployer), req)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// unauthenticated -> unauthenticated
	_, err = srv.RaiseContest(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestContestServer_RaiseInvalidArgument(t *testing.T) {
	srv := contestServer()
	ctx := asRole(context.Background(), identity.RoleCandidate)
	// blank reason -> invalid argument
	_, err := srv.RaiseContest(ctx, &caliberv1.RaiseContestRequest{
		Subject: caliberv1.ContestSubject_CONTEST_SUBJECT_MATCH, SubjectId: kernel.NewID().String(), Reason: "  ",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	// unspecified subject -> invalid argument
	_, err = srv.RaiseContest(ctx, &caliberv1.RaiseContestRequest{
		Subject: caliberv1.ContestSubject_CONTEST_SUBJECT_UNSPECIFIED, SubjectId: kernel.NewID().String(), Reason: "r",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestContestServer_ResolveAsReviewer(t *testing.T) {
	srv := contestServer()
	candCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleCandidate.String()})
	raised, err := srv.RaiseContest(candCtx, &caliberv1.RaiseContestRequest{
		Subject: caliberv1.ContestSubject_CONTEST_SUBJECT_REPORT_CARD, SubjectId: kernel.NewID().String(), Reason: "evidence misquoted",
	})
	require.NoError(t, err)

	resolved, err := srv.ResolveContest(asRole(context.Background(), identity.RoleEmployer),
		&caliberv1.ResolveContestRequest{ContestId: raised.GetContest().GetId(), Uphold: true, Note: "agreed; rescoring"})
	require.NoError(t, err)
	assert.Equal(t, caliberv1.ContestStatus_CONTEST_STATUS_UPHELD, resolved.GetContest().GetStatus())
	assert.Equal(t, "agreed; rescoring", resolved.GetContest().GetResolution())

	// a candidate cannot resolve
	_, err = srv.ResolveContest(asRole(context.Background(), identity.RoleCandidate),
		&caliberv1.ResolveContestRequest{ContestId: raised.GetContest().GetId(), Uphold: true})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}
