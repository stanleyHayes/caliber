package grpcadapter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

func ctxWithBearer(token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestAuthInterceptorInjectsPrincipal(t *testing.T) {
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenService(ctrl)
	want := app.Principal{UserID: kernel.NewID(), Role: "employer"}
	tokens.EXPECT().VerifyAccess("good").Return(want, nil)

	var seen app.Principal
	var ok bool
	handler := func(ctx context.Context, _ any) (any, error) {
		seen, ok = PrincipalFromContext(ctx)
		return "ok", nil
	}
	_, err := NewAuthInterceptor(tokens)(ctxWithBearer("good"), nil, &grpc.UnaryServerInfo{}, handler)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, want, seen)
}

func TestAuthInterceptorRejectsInvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenService(ctrl)
	tokens.EXPECT().VerifyAccess("bad").Return(app.Principal{}, kernel.Unauthorized("nope"))

	called := false
	handler := func(context.Context, any) (any, error) { called = true; return nil, nil }
	_, err := NewAuthInterceptor(tokens)(ctxWithBearer("bad"), nil, &grpc.UnaryServerInfo{}, handler)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.False(t, called, "handler not reached on invalid token")
}

func TestAuthInterceptorAnonymousWhenNoToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	tokens := mocks.NewMockTokenService(ctrl) // VerifyAccess never called

	handler := func(ctx context.Context, _ any) (any, error) {
		_, ok := PrincipalFromContext(ctx)
		assert.False(t, ok, "no principal when no token")
		return "ok", nil
	}
	_, err := NewAuthInterceptor(tokens)(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	require.NoError(t, err)
}

func TestRequireRole(t *testing.T) {
	base := context.Background()
	withEmployer := context.WithValue(base, principalKey{}, app.Principal{UserID: kernel.NewID(), Role: "employer"})

	t.Run("allowed role passes", func(t *testing.T) {
		_, err := RequireRole(withEmployer, identity.RoleEmployer, identity.RoleRecruiter)
		require.NoError(t, err)
	})
	t.Run("wrong role is forbidden", func(t *testing.T) {
		_, err := RequireRole(withEmployer, identity.RoleCandidate)
		assert.Equal(t, kernel.KindForbidden, kernel.KindOf(err))
	})
	t.Run("anonymous is unauthorized", func(t *testing.T) {
		_, err := RequireRole(base, identity.RoleEmployer)
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
}

func TestGetMeWithPrincipal(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, d := newIdentityServer(t, ctrl)
	u := activeUserForHandler(t)
	d.users.EXPECT().ByID(gomock.Any(), u.ID).Return(u, nil)

	ctx := context.WithValue(context.Background(), principalKey{}, app.Principal{UserID: u.ID, Role: "employer"})
	resp, err := srv.GetMe(ctx, &caliberv1.GetMeRequest{})
	require.NoError(t, err)
	assert.Equal(t, u.Email.String(), resp.GetUser().GetEmail())
}
