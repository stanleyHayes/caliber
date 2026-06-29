package grpcadapter

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
	"github.com/xcreativs/caliber/internal/domain/identity"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// PrivacyServer implements caliberv1.PrivacyServiceServer (CAL-118): data subject
// rights. The acting subject is always the authenticated candidate from the
// access token, never a request-body id — so a candidate can only export their
// own data.
type PrivacyServer struct {
	caliberv1.UnimplementedPrivacyServiceServer

	exporter *privacyapp.Exporter
	eraser   *privacyapp.Eraser
}

// NewPrivacyServer builds the privacy gRPC service over the DSAR export + erasure
// use-cases.
func NewPrivacyServer(exporter *privacyapp.Exporter, eraser *privacyapp.Eraser) *PrivacyServer {
	return &PrivacyServer{exporter: exporter, eraser: eraser}
}

// ExportMyData returns the authenticated candidate's complete data export (Ghana
// DPA 2012, right of access) as a JSON document.
func (s *PrivacyServer) ExportMyData(
	ctx context.Context, _ *caliberv1.ExportMyDataRequest,
) (*caliberv1.ExportMyDataResponse, error) {
	principal, err := RequireRole(ctx, identity.RoleCandidate)
	if err != nil {
		return nil, errToStatus(err)
	}
	export, err := s.exporter.ExportCandidate(ctx, principal.UserID)
	if err != nil {
		return nil, errToStatus(err)
	}
	doc, err := export.JSON()
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.ExportMyDataResponse{Document: string(doc)}, nil
}

// DeleteMyData erases the authenticated candidate's data (Ghana DPA 2012, right
// to erasure): a hard-delete cascade across their records, the audit trail
// retained but de-identified. Candidate-self only.
func (s *PrivacyServer) DeleteMyData(
	ctx context.Context, _ *caliberv1.DeleteMyDataRequest,
) (*caliberv1.DeleteMyDataResponse, error) {
	principal, err := RequireRole(ctx, identity.RoleCandidate)
	if err != nil {
		return nil, errToStatus(err)
	}
	if s.eraser == nil {
		return nil, status.Error(codes.Unimplemented, "privacy: data erasure is not available in this environment")
	}
	if err := s.eraser.EraseCandidate(ctx, principal.UserID); err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.DeleteMyDataResponse{}, nil
}
