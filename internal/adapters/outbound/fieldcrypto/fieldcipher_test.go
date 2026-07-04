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
	_, err = fieldcrypto.NewFieldCipher(testKey(t), "not-base64!!")
	require.Error(t, err, "an invalid previous key is rejected too")
}

// TestFieldCipher_RotationReadsBothGenerations proves the zero-downtime rotation
// property: a cipher configured with a new primary and the old key as previous
// decrypts data sealed by EITHER key, while new writes use the new key (CAL-117).
func TestFieldCipher_RotationReadsBothGenerations(t *testing.T) {
	oldKey, newKey := testKey(t), testKey(t)

	oldCipher, err := fieldcrypto.NewFieldCipher(oldKey)
	require.NoError(t, err)
	oldCiphertext, err := oldCipher.Encrypt("sealed with the old key")
	require.NoError(t, err)

	// Rotation config: new key is primary, old key is a decrypt-only fallback.
	rotating, err := fieldcrypto.NewFieldCipher(newKey, oldKey)
	require.NoError(t, err)

	// It reads the old-key ciphertext...
	dec, err := rotating.Decrypt(oldCiphertext)
	require.NoError(t, err)
	assert.Equal(t, "sealed with the old key", dec)

	// ...and its own new-key writes.
	newCiphertext, err := rotating.Encrypt("sealed with the new key")
	require.NoError(t, err)
	dec2, err := rotating.Decrypt(newCiphertext)
	require.NoError(t, err)
	assert.Equal(t, "sealed with the new key", dec2)

	// New writes are sealed with the NEW key: a new-key-only cipher can read them.
	newOnly, err := fieldcrypto.NewFieldCipher(newKey)
	require.NoError(t, err)
	dec3, err := newOnly.Decrypt(newCiphertext)
	require.NoError(t, err)
	assert.Equal(t, "sealed with the new key", dec3)
}

// TestFieldCipher_PreviousOnlyStillDecrypts guards against a botched rotation: a
// cipher with NO primary key but a previous key set must still decrypt old-key
// data rather than hand back ciphertext as if it were plaintext.
func TestFieldCipher_PreviousOnlyStillDecrypts(t *testing.T) {
	oldKey := testKey(t)
	oldCipher, err := fieldcrypto.NewFieldCipher(oldKey)
	require.NoError(t, err)
	ct, err := oldCipher.Encrypt("candidate PII")
	require.NoError(t, err)

	// Primary cleared, old key left as previous (an operator slip).
	misconfigured, err := fieldcrypto.NewFieldCipher("", oldKey)
	require.NoError(t, err)
	assert.False(t, misconfigured.Enabled(), "no primary => writes are passthrough")
	dec, err := misconfigured.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "candidate PII", dec, "a previous key still decrypts even without a primary")
}

// TestFieldCipher_RetiredKeyCannotDecrypt: once the old key is dropped from the
// config, data still sealed with it can no longer be read — which is exactly why
// the reencrypt command must rewrite every row before retiring a key.
func TestFieldCipher_RetiredKeyCannotDecrypt(t *testing.T) {
	oldKey, newKey := testKey(t), testKey(t)
	oldCipher, err := fieldcrypto.NewFieldCipher(oldKey)
	require.NoError(t, err)
	oldCiphertext, err := oldCipher.Encrypt("still on the old key")
	require.NoError(t, err)

	newOnly, err := fieldcrypto.NewFieldCipher(newKey) // old key retired
	require.NoError(t, err)
	_, err = newOnly.Decrypt(oldCiphertext)
	require.Error(t, err, "a value sealed by a retired key is unreadable")
}
