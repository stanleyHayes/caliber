package identity_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

type fakeSSO struct {
	identity app.SSOIdentity
	err      error
}

func (f fakeSSO) Authenticate(context.Context, string) (app.SSOIdentity, error) {
	return f.identity, f.err
}

func ssoService(d deps, a app.SSOAuthenticator) *identityapp.Service {
	return identityapp.NewService(d.users, d.hasher, d.tokens, d.refresh,
		func() time.Time { return time.Unix(1700000000, 0) }, identityapp.WithSSO(a))
}

// TestLoginWithSSO_Success proves the pluggability seam (CAL-155): an IdP-asserted
// identity is mapped to the pre-registered account and a normal Caliber session
// is issued via the same TokenService as a password login.
func TestLoginWithSSO_Success(t *testing.T) {
	d := newDeps(gomock.NewController(t))
	svc := ssoService(d, fakeSSO{identity: app.SSOIdentity{Subject: "okta|1", Email: "ama@example.com", Name: "Ama"}})
	d.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(activeUser(t), nil)
	d.expectIssue()

	sess, err := svc.LoginWithSSO(context.Background(), "id-token")
	require.NoError(t, err)
	assert.Equal(t, "access", sess.Access.Token)
}

// TestLoginWithSSO_NotConfigured: without a wired authenticator, SSO is inert.
func TestLoginWithSSO_NotConfigured(t *testing.T) {
	d := newDeps(gomock.NewController(t))
	_, err := d.service().LoginWithSSO(context.Background(), "tok")
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

// TestLoginWithSSO_UnknownAccount: the POC does no just-in-time provisioning, so
// an asserted identity with no local account is rejected (generic unauthorized).
func TestLoginWithSSO_UnknownAccount(t *testing.T) {
	d := newDeps(gomock.NewController(t))
	svc := ssoService(d, fakeSSO{identity: app.SSOIdentity{Email: "ghost@example.com"}})
	d.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("no user"))

	_, err := svc.LoginWithSSO(context.Background(), "tok")
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

// TestLoginWithSSO_AuthenticatorRejects: a bad/expired assertion surfaces the
// adapter's unauthorized error and never reaches the user store.
func TestLoginWithSSO_AuthenticatorRejects(t *testing.T) {
	d := newDeps(gomock.NewController(t))
	svc := ssoService(d, fakeSSO{err: kernel.Unauthorized("bad token")})

	_, err := svc.LoginWithSSO(context.Background(), "tok")
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

// TestLoginWithSSO_InvalidEmail: an assertion with no valid email is rejected.
func TestLoginWithSSO_InvalidEmail(t *testing.T) {
	d := newDeps(gomock.NewController(t))
	svc := ssoService(d, fakeSSO{identity: app.SSOIdentity{Email: "not-an-email"}})

	_, err := svc.LoginWithSSO(context.Background(), "tok")
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}
