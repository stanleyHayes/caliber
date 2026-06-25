package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/app"
	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	identitydom "github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

type idDeps struct {
	users   *mocks.MockUserRepository
	hasher  *mocks.MockPasswordHasher
	tokens  *mocks.MockTokenService
	refresh *mocks.MockRefreshTokenStore
}

func newIdentityServer(t *testing.T, ctrl *gomock.Controller) (*IdentityServer, idDeps) {
	t.Helper()
	d := idDeps{
		users:   mocks.NewMockUserRepository(ctrl),
		hasher:  mocks.NewMockPasswordHasher(ctrl),
		tokens:  mocks.NewMockTokenService(ctrl),
		refresh: mocks.NewMockRefreshTokenStore(ctrl),
	}
	svc := identityapp.NewService(d.users, d.hasher, d.tokens, d.refresh, func() time.Time { return time.Unix(1700000000, 0) })
	return NewIdentityServer(svc), d
}

func (d idDeps) expectIssue() {
	d.tokens.EXPECT().IssueAccess(gomock.Any()).Return(app.AccessToken{Token: "access", ExpiresIn: 15 * time.Minute}, nil)
	d.tokens.EXPECT().IssueRefresh(gomock.Any()).Return(app.RefreshToken{Token: "refresh", ID: "jti", ExpiresIn: time.Hour}, nil)
	d.refresh.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
}

func TestIdentityRegisterHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, d := newIdentityServer(t, ctrl)
	d.hasher.EXPECT().Hash(gomock.Any()).Return("hashed", nil)
	d.users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	d.expectIssue()

	resp, err := srv.Register(context.Background(), &caliberv1.RegisterRequest{
		Email: "ama@example.com", Password: "super-secret-pass", Name: "Ama", Role: caliberv1.UserRole_USER_ROLE_EMPLOYER,
	})
	require.NoError(t, err)
	assert.Equal(t, "ama@example.com", resp.GetUser().GetEmail())
	assert.Equal(t, caliberv1.UserRole_USER_ROLE_EMPLOYER, resp.GetUser().GetRole())
	assert.Equal(t, "access", resp.GetTokens().GetAccessToken())
	assert.Equal(t, int64(900), resp.GetTokens().GetAccessExpiresIn())
}

func TestIdentityRegisterRejectsUnspecifiedRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, _ := newIdentityServer(t, ctrl)
	_, err := srv.Register(context.Background(), &caliberv1.RegisterRequest{
		Email: "ama@example.com", Password: "super-secret-pass", Name: "Ama", Role: caliberv1.UserRole_USER_ROLE_UNSPECIFIED,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestIdentityLoginHandlerUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, d := newIdentityServer(t, ctrl)
	d.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, kernelNotFound())
	_, err := srv.Login(context.Background(), &caliberv1.LoginRequest{Email: "x@example.com", Password: "whatever-secret"})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestIdentityRefreshHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, d := newIdentityServer(t, ctrl)
	u := activeUserForHandler(t)
	d.tokens.EXPECT().VerifyRefresh("rt").Return(app.RefreshClaims{Principal: app.Principal{UserID: u.ID, Role: "employer"}, ID: "old"}, nil)
	d.refresh.EXPECT().Consume(gomock.Any(), "old", gomock.Any()).Return(app.RefreshRecord{ID: "old", UserID: u.ID}, nil)
	d.users.EXPECT().ByID(gomock.Any(), u.ID).Return(u, nil)
	d.expectIssue()

	resp, err := srv.Refresh(context.Background(), &caliberv1.RefreshRequest{RefreshToken: "rt"})
	require.NoError(t, err)
	assert.Equal(t, "refresh", resp.GetTokens().GetRefreshToken())
}

func TestIdentityLogoutHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, d := newIdentityServer(t, ctrl)
	d.tokens.EXPECT().VerifyRefresh("rt").Return(app.RefreshClaims{ID: "jti"}, nil)
	d.refresh.EXPECT().Revoke(gomock.Any(), "jti").Return(nil)
	_, err := srv.Logout(context.Background(), &caliberv1.LogoutRequest{RefreshToken: "rt"})
	require.NoError(t, err)
}

func TestIdentityGetMeRequiresAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, _ := newIdentityServer(t, ctrl)
	_, err := srv.GetMe(context.Background(), &caliberv1.GetMeRequest{})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func kernelNotFound() error { return kernel.NotFound("memory: user not found") }

func activeUserForHandler(t *testing.T) *identitydom.User {
	t.Helper()
	e, err := identitydom.NewEmail("ama@example.com")
	require.NoError(t, err)
	u, err := identitydom.NewUser(e, identitydom.RoleEmployer, "Ama", "hashed", time.Unix(1700000000, 0))
	require.NoError(t, err)
	return u
}
