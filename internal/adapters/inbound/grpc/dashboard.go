package grpcadapter

import (
	"context"

	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// DashboardServer implements caliberv1.DashboardServiceServer (Talent Radar).
type DashboardServer struct {
	caliberv1.UnimplementedDashboardServiceServer

	agg *dashboardapp.Aggregator
}

// NewDashboardServer builds the dashboard gRPC service from its read model.
func NewDashboardServer(agg *dashboardapp.Aggregator) *DashboardServer {
	return &DashboardServer{agg: agg}
}

// GetPool returns the paginated live candidate pool.
func (s *DashboardServer) GetPool(ctx context.Context, req *caliberv1.GetPoolRequest) (*caliberv1.GetPoolResponse, error) {
	page := pageFromProto(req.GetPage())
	pool, total, err := s.agg.Pool(ctx, page)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.PoolCandidate, 0, len(pool))
	for _, c := range pool {
		out = append(out, &caliberv1.PoolCandidate{
			CandidateId:    c.CandidateID.String(),
			Name:           c.Name,
			PassportStatus: passportStatusToProto(c.PassportStatus),
			HeadlineScore:  c.HeadlineScore,
		})
	}
	return &caliberv1.GetPoolResponse{Candidates: out, Page: pageResponseToProto(page, total)}, nil
}

// GetSupplyDemand returns the open-roles-vs-pool snapshot by role family.
func (s *DashboardServer) GetSupplyDemand(
	ctx context.Context, _ *caliberv1.GetSupplyDemandRequest,
) (*caliberv1.GetSupplyDemandResponse, error) {
	items, err := s.agg.SupplyDemand(ctx)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.SupplyDemandItem, 0, len(items))
	for _, it := range items {
		out = append(out, &caliberv1.SupplyDemandItem{
			RoleFamily:          it.RoleFamily,
			OpenRoles:           int32(it.OpenRoles),           //nolint:gosec // small bounded counts
			AvailableCandidates: int32(it.AvailableCandidates), //nolint:gosec // small bounded counts
			Gap:                 int32(it.Gap),                 //nolint:gosec // small bounded counts
		})
	}
	return &caliberv1.GetSupplyDemandResponse{Items: out}, nil
}

// GetAlerts returns the paginated two-way match alert feed.
func (s *DashboardServer) GetAlerts(ctx context.Context, req *caliberv1.GetAlertsRequest) (*caliberv1.GetAlertsResponse, error) {
	page := pageFromProto(req.GetPage())
	alerts, total, err := s.agg.Alerts(ctx, page)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.MatchAlert, 0, len(alerts))
	for _, al := range alerts {
		out = append(out, &caliberv1.MatchAlert{
			Id:          al.ID.String(),
			RoleId:      al.RoleID.String(),
			CandidateId: al.CandidateID.String(),
			Message:     al.Message,
		})
	}
	return &caliberv1.GetAlertsResponse{Alerts: out, Page: pageResponseToProto(page, total)}, nil
}

// GetTimeToShortlist returns the headline weeks-to-hours metric.
func (s *DashboardServer) GetTimeToShortlist(
	ctx context.Context, _ *caliberv1.GetTimeToShortlistRequest,
) (*caliberv1.GetTimeToShortlistResponse, error) {
	m := s.agg.TimeToShortlist(ctx)
	return &caliberv1.GetTimeToShortlistResponse{Metric: &caliberv1.TimeToShortlist{
		BaselineHours:     m.BaselineHours,
		CurrentHours:      m.CurrentHours,
		ImprovementFactor: m.ImprovementFactor,
	}}, nil
}

func passportStatusToProto(s talent.PassportStatus) caliberv1.PassportStatus {
	switch s {
	case talent.PassportCVOnly:
		return caliberv1.PassportStatus_PASSPORT_STATUS_CV_ONLY
	case talent.PassportScreened:
		return caliberv1.PassportStatus_PASSPORT_STATUS_SCREENED
	case talent.PassportVerified:
		return caliberv1.PassportStatus_PASSPORT_STATUS_VERIFIED
	default:
		return caliberv1.PassportStatus_PASSPORT_STATUS_UNSPECIFIED
	}
}
