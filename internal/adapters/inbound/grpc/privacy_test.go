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

	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
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

	srv := NewPrivacyServer(privacyapp.NewExporter(candidates, profiles, apps, interviews, contests))
	// The acting subject comes from the auth context — the candidate exports their own data.
	resp, err := srv.ExportMyData(asCandidate(context.Background(), cid), &caliberv1.ExportMyDataRequest{})
	require.NoError(t, err)

	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(resp.GetDocument()), &doc))
	assert.Contains(t, doc, "candidate")
	assert.Contains(t, doc, "applications")
}

func TestExportMyData_RequiresCandidate(t *testing.T) {
	srv := NewPrivacyServer(nil)
	// A reviewer is not a data subject here.
	_, err := srv.ExportMyData(asRole(context.Background(), identity.RoleEmployer), &caliberv1.ExportMyDataRequest{})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	// Unauthenticated is rejected.
	_, err = srv.ExportMyData(context.Background(), &caliberv1.ExportMyDataRequest{})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
