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
	identitydom "github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/mocks"
)

type deps struct {
	users   *mocks.MockUserRepository
	hasher  *mocks.MockPasswordHasher
	tokens  *mocks.MockTokenService
	refresh *mocks.MockRefreshTokenStore
}

func newDeps(ctrl *gomock.Controller) deps {
	return deps{
		users:   mocks.NewMockUserRepository(ctrl),
		hasher:  mocks.NewMockPasswordHasher(ctrl),
		tokens:  mocks.NewMockTokenService(ctrl),
		refresh: mocks.NewMockRefreshTokenStore(ctrl),
	}
}

func (d deps) service() *identityapp.Service {
	return identityapp.NewService(d.users, d.hasher, d.tokens, d.refresh, func() time.Time { return time.Unix(1700000000, 0) })
}

// expectIssue sets up the token issuance + refresh-save that every successful
// authentication performs.
func (d deps) expectIssue() {
	d.tokens.EXPECT().IssueAccess(gomock.Any()).Return(app.AccessToken{Token: "access", ExpiresIn: 15 * time.Minute}, nil)
	d.tokens.EXPECT().IssueRefresh(gomock.Any()).Return(app.RefreshToken{Token: "refresh", ID: "jti", ExpiresIn: time.Hour}, nil)
	d.refresh.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
}

func activeUser(t *testing.T) *identitydom.User {
	t.Helper()
	email, err := identitydom.NewEmail("ama@example.com")
	require.NoError(t, err)
	u, err := identitydom.NewUser(email, identitydom.RoleEmployer, "Ama", "hashed-pw", time.Unix(1700000000, 0))
	require.NoError(t, err)
	return u
}

func TestRegisterSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.hasher.EXPECT().Hash("super-secret-pass").Return("hashed-pw", nil)
	d.users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	d.expectIssue()

	sess, err := d.service().Register(context.Background(), identityapp.RegisterInput{
		Email: "Ama@Example.com", Password: "super-secret-pass", Name: "Ama", Role: identitydom.RoleEmployer,
	})
	require.NoError(t, err)
	assert.Equal(t, identitydom.Email("ama@example.com"), sess.User.Email)
	assert.Equal(t, "access", sess.Access.Token)
	assert.Equal(t, "jti", sess.Refresh.ID)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.hasher.EXPECT().Hash(gomock.Any()).Return("hashed-pw", nil)
	d.users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(kernel.Conflict("dup"))

	_, err := d.service().Register(context.Background(), identityapp.RegisterInput{
		Email: "ama@example.com", Password: "super-secret-pass", Name: "Ama", Role: identitydom.RoleEmployer,
	})
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(err))
}

func TestRegisterRejectsWeakPasswordAndBadInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl) // no mock calls expected: validation fails first
	svc := d.service()

	_, err := svc.Register(context.Background(), identityapp.RegisterInput{
		Email: "ama@example.com", Password: "short", Name: "Ama", Role: identitydom.RoleEmployer,
	})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err), "short password rejected")

	_, err = svc.Register(context.Background(), identityapp.RegisterInput{
		Email: "not-an-email", Password: "super-secret-pass", Name: "Ama", Role: identitydom.RoleEmployer,
	})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err), "bad email rejected")

	_, err = svc.Register(context.Background(), identityapp.RegisterInput{
		Email: "ama@example.com", Password: "super-secret-pass", Name: "Ama", Role: identitydom.RoleUnspecified,
	})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err), "unspecified role rejected")
}

func TestLoginSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.users.EXPECT().ByEmail(gomock.Any(), identitydom.Email("ama@example.com")).Return(activeUser(t), nil)
	d.hasher.EXPECT().Verify("hashed-pw", "super-secret-pass").Return(true, nil)
	d.expectIssue()

	sess, err := d.service().Login(context.Background(), "ama@example.com", "super-secret-pass")
	require.NoError(t, err)
	assert.Equal(t, "access", sess.Access.Token)
}

func TestLoginGenericFailures(t *testing.T) {
	t.Run("wrong password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		d.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(activeUser(t), nil)
		d.hasher.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(false, nil)
		_, err := d.service().Login(context.Background(), "ama@example.com", "nope")
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
	t.Run("unknown email", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		d.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
		_, err := d.service().Login(context.Background(), "ghost@example.com", "whatever-secret")
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
	t.Run("locked account", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		u := activeUser(t)
		u.Lock()
		d.users.EXPECT().ByEmail(gomock.Any(), gomock.Any()).Return(u, nil)
		d.hasher.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(true, nil)
		_, err := d.service().Login(context.Background(), "ama@example.com", "super-secret-pass")
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
	t.Run("malformed email", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl) // no repo call: email parse fails first
		_, err := d.service().Login(context.Background(), "bad", "whatever-secret")
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
}

func TestRefreshRotates(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	u := activeUser(t)
	d.tokens.EXPECT().VerifyRefresh("rt").Return(app.RefreshClaims{
		Principal: app.Principal{UserID: u.ID, Role: "employer"}, ID: "old-jti"}, nil)
	d.refresh.EXPECT().Consume(gomock.Any(), "old-jti", gomock.Any()).Return(app.RefreshRecord{ID: "old-jti", UserID: u.ID}, nil)
	d.users.EXPECT().ByID(gomock.Any(), u.ID).Return(u, nil)
	d.expectIssue()

	sess, err := d.service().Refresh(context.Background(), "rt")
	require.NoError(t, err)
	assert.Equal(t, "access", sess.Access.Token)
}

func TestRefreshFailures(t *testing.T) {
	t.Run("invalid token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		d.tokens.EXPECT().VerifyRefresh(gomock.Any()).Return(app.RefreshClaims{}, kernel.Unauthorized("bad"))
		_, err := d.service().Refresh(context.Background(), "rt")
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
	t.Run("replayed/consumed token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		d.tokens.EXPECT().VerifyRefresh(gomock.Any()).Return(app.RefreshClaims{ID: "jti"}, nil)
		d.refresh.EXPECT().Consume(gomock.Any(), "jti", gomock.Any()).Return(app.RefreshRecord{}, kernel.Unauthorized("revoked"))
		_, err := d.service().Refresh(context.Background(), "rt")
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
	t.Run("user gone", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		d.tokens.EXPECT().VerifyRefresh(gomock.Any()).Return(app.RefreshClaims{ID: "jti"}, nil)
		d.refresh.EXPECT().Consume(gomock.Any(), gomock.Any(), gomock.Any()).Return(app.RefreshRecord{ID: "jti"}, nil)
		d.users.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("gone"))
		_, err := d.service().Refresh(context.Background(), "rt")
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	})
}

func TestLogout(t *testing.T) {
	t.Run("revokes a valid token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		d.tokens.EXPECT().VerifyRefresh("rt").Return(app.RefreshClaims{ID: "jti"}, nil)
		d.refresh.EXPECT().Revoke(gomock.Any(), "jti").Return(nil)
		require.NoError(t, d.service().Logout(context.Background(), "rt"))
	})
	t.Run("idempotent for an invalid token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		d.tokens.EXPECT().VerifyRefresh(gomock.Any()).Return(app.RefreshClaims{}, kernel.Unauthorized("bad"))
		require.NoError(t, d.service().Logout(context.Background(), "rt"), "no error, nothing to revoke")
	})
}

func TestMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	u := activeUser(t)
	d.users.EXPECT().ByID(gomock.Any(), u.ID).Return(u, nil)
	got, err := d.service().Me(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.Email, got.Email)
}
