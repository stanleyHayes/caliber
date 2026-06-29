package grpcadapter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

func TestExportMyData_ReturnsTheCallersOwnData(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	apps := mocks.NewMockApplicationRepository(ctrl)
	interviews := mocks.NewMockInterviewRepository(ctrl)
	contests := mocks.NewMockContestRepository(ctrl)

	cid := kernel.NewID()
	cand, err := talent.NewCandidate(cid, "Accra", talent.CandidateIntake{Location: "Accra"})
	require.NoError(t, err)
	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	apps.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)
	interviews.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)
	contests.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)

	srv := NewPrivacyServer(privacyapp.NewExporter(candidates, profiles, apps, interviews, contests), nil)
	// The acting subject comes from the auth context — the candidate exports their own data.
	resp, err := srv.ExportMyData(asCandidate(context.Background(), cid), &caliberv1.ExportMyDataRequest{})
	require.NoError(t, err)

	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(resp.GetDocument()), &doc))
	assert.Contains(t, doc, "candidate")
	assert.Contains(t, doc, "applications")
}

func TestExportMyData_RequiresCandidate(t *testing.T) {
	srv := NewPrivacyServer(nil, nil)
	// A reviewer is not a data subject here.
	_, err := srv.ExportMyData(asRole(context.Background(), identity.RoleEmployer), &caliberv1.ExportMyDataRequest{})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	// Unauthenticated is rejected.
	_, err = srv.ExportMyData(context.Background(), &caliberv1.ExportMyDataRequest{})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestDeleteMyData_ErasesTheCallersData(t *testing.T) {
	ctx := context.Background()
	candidates := memory.NewCandidateRepo()
	profiles := memory.NewTalentProfileRepo()
	apps := memory.NewApplicationRepo()
	interviews := memory.NewInterviewRepo()
	matches := memory.NewMatchRepo()
	contests := memory.NewContestRepo()
	users := memory.NewUserRepo()
	auditRepo := memory.NewAuditRepo()

	cid := kernel.NewID()
	cand, err := talent.NewCandidate(cid, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NoError(t, candidates.Create(ctx, cand))
	require.NoError(t, apps.Create(ctx, &agentdom.Application{ID: kernel.NewID(), CandidateID: cid}))

	eraser := privacyapp.NewEraser(candidates, users, auditRepo, profiles, apps, interviews, matches, contests)
	srv := NewPrivacyServer(nil, eraser)

	_, err = srv.DeleteMyData(asCandidate(ctx, cid), &caliberv1.DeleteMyDataRequest{})
	require.NoError(t, err)

	// The candidate and their records are gone.
	_, err = candidates.ByID(ctx, cid)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	_, total, err := apps.ByCandidate(ctx, cid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
}

func TestDeleteMyData_RequiresCandidateAndAvailableEraser(t *testing.T) {
	// Reviewer / anonymous are rejected before erasure.
	srv := NewPrivacyServer(nil, nil)
	_, err := srv.DeleteMyData(asRole(context.Background(), identity.RoleEmployer), &caliberv1.DeleteMyDataRequest{})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = srv.DeleteMyData(context.Background(), &caliberv1.DeleteMyDataRequest{})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	// A candidate caller, but no eraser wired (e.g. Postgres path) -> Unimplemented.
	_, err = srv.DeleteMyData(asCandidate(context.Background(), kernel.NewID()), &caliberv1.DeleteMyDataRequest{})
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}
