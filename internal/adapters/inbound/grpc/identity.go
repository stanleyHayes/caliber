package grpcadapter

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// IdentityServer implements caliberv1.IdentityServiceServer (EPIC-02 auth).
type IdentityServer struct {
	caliberv1.UnimplementedIdentityServiceServer

	svc *identityapp.Service
}

// NewIdentityServer builds the identity gRPC service from its use-case.
func NewIdentityServer(svc *identityapp.Service) *IdentityServer { return &IdentityServer{svc: svc} }

// Register creates an account and returns the user with a fresh token pair.
func (s *IdentityServer) Register(ctx context.Context, req *caliberv1.RegisterRequest) (*caliberv1.RegisterResponse, error) {
	role, err := userRoleFromProto(req.GetRole())
	if err != nil {
		return nil, errToStatus(err)
	}
	sess, err := s.svc.Register(ctx, identityapp.RegisterInput{
		Email: req.GetEmail(), Password: req.GetPassword(), Name: req.GetName(), Role: role,
	})
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.RegisterResponse{User: userToProto(sess.User), Tokens: tokenPairToProto(sess)}, nil
}

// Login verifies credentials and returns the user with a fresh token pair.
func (s *IdentityServer) Login(ctx context.Context, req *caliberv1.LoginRequest) (*caliberv1.LoginResponse, error) {
	sess, err := s.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.LoginResponse{User: userToProto(sess.User), Tokens: tokenPairToProto(sess)}, nil
}

// Refresh rotates the refresh token and returns a new token pair.
func (s *IdentityServer) Refresh(ctx context.Context, req *caliberv1.RefreshRequest) (*caliberv1.RefreshResponse, error) {
	sess, err := s.svc.Refresh(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.RefreshResponse{Tokens: tokenPairToProto(sess)}, nil
}

// Logout revokes the presented refresh token (idempotent).
func (s *IdentityServer) Logout(ctx context.Context, req *caliberv1.LogoutRequest) (*caliberv1.LogoutResponse, error) {
	if err := s.svc.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.LogoutResponse{}, nil
}

// GetMe returns the authenticated user. It requires the principal injected by
// the auth interceptor (CAL-021); until that lands it reports unauthenticated.
func (s *IdentityServer) GetMe(_ context.Context, _ *caliberv1.GetMeRequest) (*caliberv1.GetMeResponse, error) {
	return nil, status.Error(codes.Unauthenticated, "identity: authentication required")
}

func userRoleFromProto(r caliberv1.UserRole) (identity.Role, error) {
	switch r {
	case caliberv1.UserRole_USER_ROLE_EMPLOYER:
		return identity.RoleEmployer, nil
	case caliberv1.UserRole_USER_ROLE_RECRUITER:
		return identity.RoleRecruiter, nil
	case caliberv1.UserRole_USER_ROLE_CANDIDATE:
		return identity.RoleCandidate, nil
	default:
		return identity.RoleUnspecified, kernel.Invalid("identity: a valid role is required")
	}
}

func userRoleToProto(r identity.Role) caliberv1.UserRole {
	switch r {
	case identity.RoleEmployer:
		return caliberv1.UserRole_USER_ROLE_EMPLOYER
	case identity.RoleRecruiter:
		return caliberv1.UserRole_USER_ROLE_RECRUITER
	case identity.RoleCandidate:
		return caliberv1.UserRole_USER_ROLE_CANDIDATE
	default:
		return caliberv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func userToProto(u *identity.User) *caliberv1.User {
	return &caliberv1.User{
		Id:        u.ID.String(),
		Email:     u.Email.String(),
		Role:      userRoleToProto(u.Role),
		Name:      u.Name,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}

func tokenPairToProto(sess *identityapp.Session) *caliberv1.TokenPair {
	return &caliberv1.TokenPair{
		AccessToken:     sess.Access.Token,
		RefreshToken:    sess.Refresh.Token,
		AccessExpiresIn: int64(sess.Access.ExpiresIn.Seconds()),
	}
}
