package grpcadapter

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

type principalKey struct{}

const bearerPrefix = "bearer "

// NewAuthInterceptor returns a unary interceptor that authenticates requests
// carrying an "authorization: Bearer <access-token>" header: a valid token's
// principal is injected into the context for downstream handlers and RBAC
// guards. A present-but-invalid token is rejected (401); an absent token leaves
// the request anonymous, and each protected handler enforces presence/role via
// PrincipalFromContext / RequireRole.
func NewAuthInterceptor(verifier app.TokenService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		raw, ok := bearerFromContext(ctx)
		if ok {
			principal, err := verifier.VerifyAccess(raw)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "auth: invalid or expired access token")
			}
			ctx = context.WithValue(ctx, principalKey{}, principal)
		}
		return handler(ctx, req)
	}
}

// bearerFromContext extracts a bearer access token from request metadata.
func bearerFromContext(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	for _, v := range md.Get("authorization") {
		if len(v) > len(bearerPrefix) && strings.EqualFold(v[:len(bearerPrefix)], bearerPrefix) {
			return strings.TrimSpace(v[len(bearerPrefix):]), true
		}
	}
	return "", false
}

// PrincipalFromContext returns the authenticated principal injected by the auth
// interceptor, if the request carried a valid access token.
func PrincipalFromContext(ctx context.Context) (app.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(app.Principal)
	return p, ok
}

// RequireAuth returns the authenticated principal or a kernel.Unauthorized error.
func RequireAuth(ctx context.Context) (app.Principal, error) {
	p, ok := PrincipalFromContext(ctx)
	if !ok {
		return app.Principal{}, kernel.Unauthorized("auth: authentication required")
	}
	return p, nil
}

// RequireRole enforces that the authenticated principal holds one of the allowed
// roles. It returns kernel.Unauthorized when unauthenticated and kernel.Forbidden
// when authenticated but not permitted.
func RequireRole(ctx context.Context, allowed ...identity.Role) (app.Principal, error) {
	p, err := RequireAuth(ctx)
	if err != nil {
		return app.Principal{}, err
	}
	for _, role := range allowed {
		if p.Role == role.String() {
			return p, nil
		}
	}
	return app.Principal{}, kernel.Forbidden("auth: insufficient permissions for this operation")
}

// requireSelfCandidate authorizes a candidate acting on their OWN data: the caller
// must be a candidate whose id matches the target candidate id, preventing one
// candidate from operating on another's agent/profile (IDOR). Registered
// candidates have candidate.ID == user.ID (the provisioner), so the principal's
// UserID is their candidate id.
func requireSelfCandidate(ctx context.Context, candidateID string) error {
	p, err := RequireRole(ctx, identity.RoleCandidate)
	if err != nil {
		return err
	}
	if p.UserID.String() != candidateID {
		return kernel.Forbidden("auth: candidates may only act on their own data")
	}
	return nil
}
