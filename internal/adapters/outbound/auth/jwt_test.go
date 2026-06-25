package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// testSecret is a >=32-byte (256-bit) HS256 key, satisfying the strength floor.
const testSecret = "test-secret-please-rotate-0123456789"

func fixedClock(ts int64) func() time.Time {
	return func() time.Time { return time.Unix(ts, 0) }
}

func testService(t *testing.T, clock func() time.Time) *auth.JWTService {
	t.Helper()
	s, err := auth.NewJWTService(auth.JWTConfig{
		Secret: testSecret, AccessTTL: 15 * time.Minute, RefreshTTL: time.Hour,
	}, auth.WithClock(clock))
	require.NoError(t, err)
	return s
}

func principal() app.Principal {
	return app.Principal{UserID: kernel.NewID(), Role: "employer"}
}

// craftHS256 signs arbitrary claims with the real test secret, to probe the
// verifier's claim checks independently of the issuer.
func craftHS256(t *testing.T, mc jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, mc).SignedString([]byte(testSecret))
	require.NoError(t, err)
	return s
}

func baseClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "u-1", "role": "employer", "typ": "access",
		"iss": "caliber", "aud": "caliber-api",
		"exp": time.Unix(1700000000+3600, 0).Unix(),
	}
}

func TestJWTAccessRoundTrip(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	p := principal()

	at, err := s.IssueAccess(p)
	require.NoError(t, err)
	assert.Equal(t, 15*time.Minute, at.ExpiresIn)

	got, err := s.VerifyAccess(at.Token)
	require.NoError(t, err)
	assert.Equal(t, p.UserID, got.UserID)
	assert.Equal(t, "employer", got.Role)
}

func TestJWTRefreshRoundTrip(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	p := principal()

	rt, err := s.IssueRefresh(p)
	require.NoError(t, err)
	assert.NotEmpty(t, rt.ID, "refresh token carries a jti")

	got, err := s.VerifyRefresh(rt.Token)
	require.NoError(t, err)
	assert.Equal(t, p.UserID, got.Principal.UserID)
	assert.Equal(t, rt.ID, got.ID)
}

func TestJWTJTIsAreDistinct(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	a, _ := s.IssueRefresh(principal())
	b, _ := s.IssueRefresh(principal())
	assert.NotEqual(t, a.ID, b.ID, "each issued token gets a fresh jti")
}

func TestJWTTokenTypesAreNotInterchangeable(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	p := principal()

	at, _ := s.IssueAccess(p)
	rt, _ := s.IssueRefresh(p)

	_, err := s.VerifyRefresh(at.Token)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "access token rejected as refresh")
	_, err = s.VerifyAccess(rt.Token)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "refresh token rejected as access")
}

func TestJWTAccessExpiredRejected(t *testing.T) {
	issuer := testService(t, fixedClock(1700000000))
	at, err := issuer.IssueAccess(principal())
	require.NoError(t, err)

	verifier := testService(t, fixedClock(1700000000+16*60)) // past the 15m access TTL
	_, err = verifier.VerifyAccess(at.Token)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestJWTRefreshExpiredRejected(t *testing.T) {
	issuer := testService(t, fixedClock(1700000000))
	rt, err := issuer.IssueRefresh(principal())
	require.NoError(t, err)

	verifier := testService(t, fixedClock(1700000000+3601)) // past the 1h refresh TTL
	_, err = verifier.VerifyRefresh(rt.Token)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestJWTMissingExpRejected(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	mc := baseClaims()
	delete(mc, "exp")
	_, err := s.VerifyAccess(craftHS256(t, mc))
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "WithExpirationRequired rejects tokens without exp")
}

func TestJWTNotYetValidRejected(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	mc := baseClaims()
	mc["nbf"] = time.Unix(1700000000+600, 0).Unix() // not valid for 10 more minutes
	_, err := s.VerifyAccess(craftHS256(t, mc))
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestJWTTamperedRejected(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	at, _ := s.IssueAccess(principal())
	tampered := at.Token[:len(at.Token)-2] + "xy"
	_, err := s.VerifyAccess(tampered)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestJWTMalformedTokenStrings(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	for _, tok := range []string{"", "abc", "a.b", "a.b.c.d", "not.a.token", "....."} {
		_, err := s.VerifyAccess(tok)
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "VerifyAccess(%q)", tok)
		_, err = s.VerifyRefresh(tok)
		assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "VerifyRefresh(%q)", tok)
	}
}

func TestJWTWrongSecretRejected(t *testing.T) {
	issuer := testService(t, fixedClock(1700000000))
	at, _ := issuer.IssueAccess(principal())

	other, err := auth.NewJWTService(
		auth.JWTConfig{Secret: "a-totally-different-secret-0123456789"}, auth.WithClock(fixedClock(1700000000)))
	require.NoError(t, err)
	_, err = other.VerifyAccess(at.Token)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestJWTWrongIssuerOrAudienceRejected(t *testing.T) {
	at, _ := testService(t, fixedClock(1700000000)).IssueAccess(principal())

	wrongIss, _ := auth.NewJWTService(auth.JWTConfig{
		Secret: testSecret, Issuer: "evil", Audience: "caliber-api",
	}, auth.WithClock(fixedClock(1700000000)))
	_, err := wrongIss.VerifyAccess(at.Token)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "issuer mismatch rejected")

	wrongAud, _ := auth.NewJWTService(auth.JWTConfig{
		Secret: testSecret, Issuer: "caliber", Audience: "someone-else",
	}, auth.WithClock(fixedClock(1700000000)))
	_, err = wrongAud.VerifyAccess(at.Token)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "audience mismatch rejected")
}

// TestJWTAlgNoneRejected proves the "none" algorithm cannot forge a token.
func TestJWTAlgNoneRejected(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	forged, err := jwt.NewWithClaims(jwt.SigningMethodNone, baseClaims()).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = s.VerifyAccess(forged)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

// TestJWTAlgConfusionRS256Rejected proves an asymmetric-signed token is rejected
// (WithValidMethods pins HS256), closing the RS256->HS256 key-confusion vector.
func TestJWTAlgConfusionRS256Rejected(t *testing.T) {
	s := testService(t, fixedClock(1700000000))
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	forged, err := jwt.NewWithClaims(jwt.SigningMethodRS256, baseClaims()).SignedString(key)
	require.NoError(t, err)
	_, err = s.VerifyAccess(forged)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestJWTEmptySecretRejected(t *testing.T) {
	_, err := auth.NewJWTService(auth.JWTConfig{Secret: "  "})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestJWTShortSecretRejected(t *testing.T) {
	_, err := auth.NewJWTService(auth.JWTConfig{Secret: "too-short"})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err), "secrets under 32 bytes are rejected")
}
