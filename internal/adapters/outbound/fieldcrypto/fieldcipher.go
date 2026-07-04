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
// The marker does not encode WHICH key sealed a value, so Decrypt authenticates
// against the primary key and then each previous key in turn (GCM's tag makes a
// wrong key fail cleanly). Key rotation is therefore live-safe: configure the new
// key as primary and the old key(s) as previous, and reads keep working while new
// writes use the new key; the `reencrypt` command then rewrites every row so the
// old key can be retired. This also resolves the passthrough->keyed transition
// (plaintext, including a value that happens to start with "enc:v1:", is
// re-encrypted into genuine ciphertext) — see docs/runbooks/key-rotation.md.
const prefix = "enc:v1:"

// FieldCipher encrypts/decrypts individual PII fields with AES-256-GCM. With no
// primary key it is a passthrough, so local/dev runs store plaintext and
// pre-existing data keeps working; configuring a key encrypts new writes while
// still reading old plaintext transparently. Optional previous keys are tried on
// decrypt only, so a key can be rotated without downtime.
type FieldCipher struct {
	primary  cipher.AEAD   // nil = passthrough (no key configured); used for Encrypt + Decrypt
	previous []cipher.AEAD // prior-generation keys, tried on Decrypt only (rotation)
}

// NewFieldCipher builds a cipher from a base64-encoded 32-byte (AES-256) primary
// key, plus any base64 previous keys that Decrypt should also accept (for a
// zero-downtime rotation). An empty primary key yields a passthrough cipher; empty
// previous keys are ignored.
func NewFieldCipher(base64Key string, previousKeys ...string) (*FieldCipher, error) {
	primary, err := newAEAD(base64Key)
	if err != nil {
		return nil, err
	}
	fc := &FieldCipher{primary: primary}
	for _, pk := range previousKeys {
		aead, perr := newAEAD(pk)
		if perr != nil {
			return nil, perr
		}
		if aead != nil {
			fc.previous = append(fc.previous, aead)
		}
	}
	return fc, nil
}

// newAEAD builds an AES-256-GCM AEAD from a base64 key, or a nil AEAD (the
// passthrough sentinel) for an empty key.
func newAEAD(base64Key string) (cipher.AEAD, error) {
	if base64Key == "" {
		return nil, nil //nolint:nilnil // nil AEAD + nil err is the intended "no key -> passthrough" signal
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
	return cipher.NewGCM(block)
}

// Passthrough returns a cipher that performs no encryption — the default when
// no key is configured.
func Passthrough() *FieldCipher { return &FieldCipher{} }

// Enabled reports whether encryption is active (a primary key was configured).
func (c *FieldCipher) Enabled() bool { return c.primary != nil }

// Encrypt returns a prefixed base64 ciphertext sealed with the primary key. A
// passthrough cipher or an empty string is returned unchanged, so an empty PII
// field stays empty/NULL rather than becoming ciphertext.
func (c *FieldCipher) Encrypt(plaintext string) (string, error) {
	if c.primary == nil || plaintext == "" {
		return plaintext, nil
	}
	nonce := make([]byte, c.primary.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := c.primary.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. A value without the prefix is returned as-is (legacy
// plaintext, or a passthrough cipher), so enabling encryption never breaks
// existing rows. A prefixed value is authenticated against the primary key and
// then each previous key; a value that no configured key can open (tampered, or
// sealed by a retired key) returns an error.
func (c *FieldCipher) Decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, prefix) {
		return stored, nil // legacy plaintext (or passthrough)
	}
	if c.primary == nil && len(c.previous) == 0 {
		// Pure passthrough: no key to try, so a prefixed value is treated as legacy
		// plaintext (the documented passthrough behavior).
		return stored, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil {
		return "", fmt.Errorf("crypto: ciphertext is not valid base64: %w", err)
	}
	// Try every configured key (primary first, then previous). This still recovers
	// data when the primary is absent but a previous key is set — a misconfigured
	// rotation must not silently hand back ciphertext as if it were plaintext.
	if c.primary != nil {
		if pt, ok := open(c.primary, raw); ok {
			return pt, nil
		}
	}
	for _, aead := range c.previous {
		if pt, ok := open(aead, raw); ok {
			return pt, nil
		}
	}
	return "", errors.New("crypto: decrypt failed (tampered, or no configured key matches)")
}

// open attempts to authenticate-and-decrypt raw (nonce||ciphertext) with aead,
// returning the plaintext and true on success, or ("", false) on any failure.
func open(aead cipher.AEAD, raw []byte) (string, bool) {
	ns := aead.NonceSize()
	if len(raw) < ns {
		return "", false
	}
	pt, err := aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", false
	}
	return string(pt), true
}
