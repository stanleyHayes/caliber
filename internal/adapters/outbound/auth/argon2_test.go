package auth_test

import (
	"encoding/base64"
	"testing"

	"github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/domain/kernel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fastParams keep Argon2id cheap for tests while exercising the same code path.
func fastParams() auth.Argon2Params {
	return auth.Argon2Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

func TestArgon2HashAndVerify(t *testing.T) {
	h := auth.NewArgon2idHasher(auth.WithArgon2Params(fastParams()))

	encoded, err := h.Hash("correct horse battery staple")
	require.NoError(t, err)
	assert.Contains(t, encoded, "$argon2id$v=19$")

	ok, err := h.Verify(encoded, "correct horse battery staple")
	require.NoError(t, err)
	assert.True(t, ok, "correct password verifies")

	bad, err := h.Verify(encoded, "wrong password")
	require.NoError(t, err)
	assert.False(t, bad, "wrong password does not verify")
}

func TestArgon2SaltIsRandom(t *testing.T) {
	h := auth.NewArgon2idHasher(auth.WithArgon2Params(fastParams()))
	a, err := h.Hash("same-password-123")
	require.NoError(t, err)
	b, err := h.Hash("same-password-123")
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "each hash uses a fresh random salt")
}

func TestArgon2VerifyAcrossParams(t *testing.T) {
	// A hash made with one parameter set verifies via a hasher with different
	// defaults, because parameters are embedded in the encoded hash.
	producer := auth.NewArgon2idHasher(auth.WithArgon2Params(
		auth.Argon2Params{Memory: 128, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32}))
	encoded, err := producer.Hash("pw-across-params")
	require.NoError(t, err)

	consumer := auth.NewArgon2idHasher(auth.WithArgon2Params(fastParams()))
	ok, err := consumer.Verify(encoded, "pw-across-params")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestArgon2HashEmptyRejected(t *testing.T) {
	_, err := auth.NewArgon2idHasher().Hash("")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestArgon2VerifyMalformed(t *testing.T) {
	h := auth.NewArgon2idHasher(auth.WithArgon2Params(fastParams()))
	cases := []string{
		"",
		"not-a-phc-string",
		"$argon2id$v=19$m=64,t=1,p=1$onlyfivefields",
		"$bcrypt$v=19$m=64,t=1,p=1$c2FsdA$a2V5",
		"$argon2id$v=99$m=64,t=1,p=1$c2FsdA$a2V5",
		"$argon2id$v=19$bad-params$c2FsdA$a2V5",
		"$argon2id$v=19$m=64,t=1,p=1$!!!notb64$a2V5",
	}
	for _, c := range cases {
		ok, err := h.Verify(c, "whatever")
		assert.False(t, ok)
		require.Error(t, err, "malformed hash %q should error", c)
	}
}

func TestArgon2VerifyRejectsDangerousParams(t *testing.T) {
	h := auth.NewArgon2idHasher(auth.WithArgon2Params(fastParams()))
	salt := base64.RawStdEncoding.EncodeToString(make([]byte, 16))
	key := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	mk := func(params string) string { return "$argon2id$v=19$" + params + "$" + salt + "$" + key }

	// t=0 and p=0 would panic argon2.IDKey; an oversized m would exhaust memory.
	for _, bad := range []string{
		mk("m=64,t=0,p=1"),
		mk("m=64,t=1,p=0"),
		mk("m=2147483647,t=1,p=1"),
		mk("m=4,t=1,p=1"), // below 8*p minimum
	} {
		ok, err := h.Verify(bad, "whatever") // must return an error, never panic
		assert.False(t, ok)
		require.Error(t, err, "dangerous params %q must be rejected", bad)
		assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
	}
}

func TestArgon2BadParamsFallBackToDefaults(t *testing.T) {
	// A hasher constructed with illegal params must not panic on Hash; it falls
	// back to safe defaults instead.
	h := auth.NewArgon2idHasher(auth.WithArgon2Params(auth.Argon2Params{Memory: 0, Iterations: 0, Parallelism: 0}))
	encoded, err := h.Hash("password-123456")
	require.NoError(t, err)
	ok, err := h.Verify(encoded, "password-123456")
	require.NoError(t, err)
	assert.True(t, ok)
}
