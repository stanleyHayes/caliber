// Package fieldcrypto provides application-level field encryption for PII at
// rest (CAL-117): AES-256-GCM over individual sensitive columns, so a database
// dump or a stolen backup does not expose candidate PII in cleartext.
package fieldcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// prefix marks a value this package encrypted. It lets Decrypt pass through
// legacy plaintext (rows written before a key was configured) and versions the
// scheme so a future key rotation can distinguish generations.
//
// Known limitation: the marker is a plaintext sentinel, so a value written while
// no key is configured that itself begins with "enc:v1:" is indistinguishable
// from real ciphertext once a key is enabled — Decrypt would try (and fail) to
// decrypt it, making that row unreadable. This only bites the passthrough->keyed
// transition on a store that already holds untrusted plaintext (e.g. a dev DB);
// in production the key is configured from the first write, so every stored
// value is genuinely encrypted and round-trips. Enabling a key on a store with
// pre-existing plaintext therefore requires a one-time re-encryption migration,
// not a live flip.
const prefix = "enc:v1:"

// FieldCipher encrypts/decrypts individual PII fields with AES-256-GCM. With no
// key it is a passthrough, so local/dev runs store plaintext and pre-existing
// data keeps working; configuring a key encrypts new writes while still reading
// old plaintext transparently.
type FieldCipher struct {
	aead cipher.AEAD // nil = passthrough (no key configured)
}

// NewFieldCipher builds a cipher from a base64-encoded 32-byte (AES-256) key.
// An empty key yields a passthrough cipher (no encryption).
func NewFieldCipher(base64Key string) (*FieldCipher, error) {
	if base64Key == "" {
		return &FieldCipher{}, nil
	}
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("crypto: field key is not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: field key must be 32 bytes (AES-256), got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &FieldCipher{aead: aead}, nil
}

// Passthrough returns a cipher that performs no encryption — the default when
// no key is configured.
func Passthrough() *FieldCipher { return &FieldCipher{} }

// Enabled reports whether encryption is active (a key was configured).
func (c *FieldCipher) Enabled() bool { return c.aead != nil }

// Encrypt returns a prefixed base64 ciphertext. A passthrough cipher or an empty
// string is returned unchanged, so an empty PII field stays empty/NULL rather
// than becoming ciphertext.
func (c *FieldCipher) Encrypt(plaintext string) (string, error) {
	if c.aead == nil || plaintext == "" {
		return plaintext, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. A value without the prefix is returned as-is (legacy
// plaintext, or a passthrough cipher), so enabling encryption never breaks
// existing rows. A tampered or wrong-key ciphertext returns an error.
func (c *FieldCipher) Decrypt(stored string) (string, error) {
	if c.aead == nil || !strings.HasPrefix(stored, prefix) {
		return stored, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil {
		return "", fmt.Errorf("crypto: ciphertext is not valid base64: %w", err)
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("crypto: ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt failed (tampered or wrong key): %w", err)
	}
	return string(pt), nil
}
