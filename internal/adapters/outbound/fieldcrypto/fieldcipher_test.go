package fieldcrypto_test

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
)

func testKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(key)
}

func TestFieldCipher_RoundTrip(t *testing.T) {
	c, err := fieldcrypto.NewFieldCipher(testKey(t))
	require.NoError(t, err)
	assert.True(t, c.Enabled())

	const plain = "Ama Mensah — backend engineer, Accra"
	enc, err := c.Encrypt(plain)
	require.NoError(t, err)
	assert.NotEqual(t, plain, enc, "ciphertext differs from plaintext")
	assert.True(t, strings.HasPrefix(enc, "enc:v1:"), "carries the scheme prefix")

	dec, err := c.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, plain, dec)

	// Nonce randomization: encrypting the same text twice yields different bytes.
	enc2, err := c.Encrypt(plain)
	require.NoError(t, err)
	assert.NotEqual(t, enc, enc2, "each encryption uses a fresh nonce")
}

func TestFieldCipher_Passthrough(t *testing.T) {
	c, err := fieldcrypto.NewFieldCipher("") // no key
	require.NoError(t, err)
	assert.False(t, c.Enabled())

	enc, err := c.Encrypt("secret")
	require.NoError(t, err)
	assert.Equal(t, "secret", enc, "passthrough stores plaintext")
	dec, err := c.Decrypt("secret")
	require.NoError(t, err)
	assert.Equal(t, "secret", dec)
}

func TestFieldCipher_EmptyStaysEmpty(t *testing.T) {
	c, err := fieldcrypto.NewFieldCipher(testKey(t))
	require.NoError(t, err)
	enc, err := c.Encrypt("")
	require.NoError(t, err)
	assert.Empty(t, enc, "an empty field is not turned into ciphertext")
}

func TestFieldCipher_ReadsLegacyPlaintext(t *testing.T) {
	// An enabled cipher must still read rows written before encryption was on.
	c, err := fieldcrypto.NewFieldCipher(testKey(t))
	require.NoError(t, err)
	dec, err := c.Decrypt("old plaintext summary")
	require.NoError(t, err)
	assert.Equal(t, "old plaintext summary", dec)
}

func TestFieldCipher_TamperDetected(t *testing.T) {
	c, err := fieldcrypto.NewFieldCipher(testKey(t))
	require.NoError(t, err)
	enc, err := c.Encrypt("evidence: shipped payments API")
	require.NoError(t, err)
	_, err = c.Decrypt(enc + "AA==") // corrupt the ciphertext tail
	require.Error(t, err, "GCM authentication rejects tampered ciphertext")
}

func TestNewFieldCipher_RejectsBadKey(t *testing.T) {
	_, err := fieldcrypto.NewFieldCipher("not-base64!!")
	require.Error(t, err)
	_, err = fieldcrypto.NewFieldCipher(base64.StdEncoding.EncodeToString([]byte("too-short")))
	require.Error(t, err, "AES-256 needs a 32-byte key")
}
