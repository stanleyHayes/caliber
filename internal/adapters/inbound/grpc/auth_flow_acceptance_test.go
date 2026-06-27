package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TestAuthFlowEndToEnd is the auth acceptance test: register -> login ->
// authenticated access, exercising the REAL Argon2id hasher and the REAL JWT
// service through the auth interceptor (no mocks). It proves the access token a
// real registration issues is verified by the interceptor and resolves the right
// principal at a protected endpoint, and that a missing/forged token is rejected.
func TestAuthFlowEndToEnd(t *testing.T) {
	ctx := context.Background()
	jwt, err := authadapter.NewJWTService(authadapter.JWTConfig{
		Secret:     "an-at-least-32-byte-test-signing-secret!!",
		Issuer:     "caliber",
		Audience:   "caliber-api",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: time.Hour,
	})
	require.NoError(t, err)
	svc := identityapp.NewService(memory.NewUserRepo(), authadapter.NewArgon2idHasher(), jwt, memory.NewRefreshStore(), time.Now)
	srv := NewIdentityServer(svc)

	const email, password = "ama@example.com", "super-secret-pass"

	// 1) Register an employer -> a user and a real access token.
	reg, err := srv.Register(ctx, &caliberv1.RegisterRequest{
		Email: email, Password: password, Name: "Ama Mensah", Role: caliberv1.UserRole_USER_ROLE_EMPLOYER,
	})
	require.NoError(t, err)
	assert.Equal(t, email, reg.GetUser().GetEmail())
	require.NotEmpty(t, reg.GetTokens().GetAccessToken())

	// 2) The access token, presented through the interceptor, authenticates a call
	//    to the protected GetMe endpoint and resolves the registered user.
	assertAuthenticatedAsAma(t, srv, jwt, reg.GetTokens().GetAccessToken(), email)

	// 3) Login with the same credentials (real Argon2id verify) -> fresh tokens
	//    that also authenticate.
	login, err := srv.Login(ctx, &caliberv1.LoginRequest{Email: email, Password: password})
	require.NoError(t, err)
	require.NotEmpty(t, login.GetTokens().GetAccessToken())
	assertAuthenticatedAsAma(t, srv, jwt, login.GetTokens().GetAccessToken(), email)

	// 4) Wrong password is rejected.
	_, err = srv.Login(ctx, &caliberv1.LoginRequest{Email: email, Password: "wrong-password"})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	// 5) A missing token is rejected at the protected endpoint.
	_, err = srv.GetMe(ctx, &caliberv1.GetMeRequest{})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	// 6) A forged/garbage token is rejected by the interceptor.
	_, err = NewAuthInterceptor(jwt)(ctxWithBearer("not-a-real-jwt"), nil, &grpc.UnaryServerInfo{},
		func(context.Context, any) (any, error) { return nil, nil })
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// assertAuthenticatedAsAma runs the access token through the auth interceptor and
// asserts the protected GetMe call (invoked with the interceptor-authenticated
// context) resolves the expected user.
func assertAuthenticatedAsAma(t *testing.T, srv *IdentityServer, jwt *authadapter.JWTService, token, email string) {
	t.Helper()
	resp, err := NewAuthInterceptor(jwt)(ctxWithBearer(token), nil, &grpc.UnaryServerInfo{},
		func(ctx context.Context, _ any) (any, error) {
			return srv.GetMe(ctx, &caliberv1.GetMeRequest{})
		})
	require.NoError(t, err)
	me, ok := resp.(*caliberv1.GetMeResponse)
	require.True(t, ok, "the protected handler returned a GetMe response")
	assert.Equal(t, email, me.GetUser().GetEmail(), "the protected endpoint resolves the authenticated user")
}
