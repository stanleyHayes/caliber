// Package auth provides outbound security adapters: Argon2id password hashing
// and JWT token issuance/verification behind the app ports.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Argon2 parameter bounds for decoding an externally-formatted hash. The lower
// bounds guard Verify against a panic (argon2.IDKey panics when t<1 or p<1); the
// upper bounds guard against a memory/CPU-exhaustion verify driven by a crafted
// or corrupted stored hash.
const (
	maxArgon2Memory      = 1 << 20 // 1 GiB expressed in KiB
	maxArgon2Iterations  = 64
	maxArgon2Parallelism = 64
)

// Argon2Params are the tunable Argon2id cost parameters.
type Argon2Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params returns OWASP-aligned Argon2id parameters
// (64 MiB, t=3, p=2, 16-byte salt, 32-byte key).
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{Memory: 64 * 1024, Iterations: 3, Parallelism: 2, SaltLength: 16, KeyLength: 32}
}

// Argon2idHasher implements app.PasswordHasher using Argon2id with PHC-encoded
// output: $argon2id$v=19$m=<KiB>,t=<iters>,p=<lanes>$<b64 salt>$<b64 key>.
type Argon2idHasher struct {
	p Argon2Params
}

// Argon2Option customizes the hasher.
type Argon2Option func(*Argon2idHasher)

// WithArgon2Params overrides the default cost parameters.
func WithArgon2Params(p Argon2Params) Argon2Option {
	return func(h *Argon2idHasher) { h.p = p }
}

// NewArgon2idHasher builds an Argon2id hasher with OWASP-aligned defaults.
func NewArgon2idHasher(opts ...Argon2Option) *Argon2idHasher {
	h := &Argon2idHasher{p: DefaultArgon2Params()}
	for _, opt := range opts {
		opt(h)
	}
	if validateArgon2Params(h.p) != nil {
		h.p = DefaultArgon2Params()
	}
	return h
}

// Hash derives an Argon2id hash with a fresh random salt and PHC-encodes it.
func (h *Argon2idHasher) Hash(plain string) (string, error) {
	if plain == "" {
		return "", kernel.Invalid("auth: cannot hash an empty password")
	}
	salt := make([]byte, h.p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", kernel.Wrap(err, kernel.KindInternal, "auth: salt generation failed")
	}
	key := argon2.IDKey([]byte(plain), salt, h.p.Iterations, h.p.Memory, h.p.Parallelism, h.p.KeyLength)
	return encodeHash(h.p, salt, key), nil
}

// Verify re-derives the hash for plain using the encoded parameters and salt and
// compares in constant time.
func (h *Argon2idHasher) Verify(encodedHash, plain string) (bool, error) {
	p, salt, key, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}
	//nolint:gosec // key length is a small bounded value (<=64), never overflows uint32
	other := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, uint32(len(key)))
	return subtle.ConstantTimeCompare(key, other) == 1, nil
}

func encodeHash(p Argon2Params, salt, key []byte) string {
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key))
}

func decodeHash(encoded string) (Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, kernel.Invalid("auth: malformed argon2 hash")
	}
	p, err := parseArgon2Params(parts[2], parts[3])
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, kernel.Invalid("auth: unreadable argon2 salt")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, kernel.Invalid("auth: unreadable argon2 key")
	}
	if len(key) == 0 {
		return Argon2Params{}, nil, nil, kernel.Invalid("auth: empty argon2 key")
	}
	//nolint:gosec // salt/key lengths are small bounded values, never overflow uint32
	p.SaltLength, p.KeyLength = uint32(len(salt)), uint32(len(key))
	return p, salt, key, nil
}

// parseArgon2Params reads and validates the version and "m=,t=,p=" segments.
func parseArgon2Params(versionPart, paramPart string) (Argon2Params, error) {
	var version int
	if _, err := fmt.Sscanf(versionPart, "v=%d", &version); err != nil {
		return Argon2Params{}, kernel.Invalid("auth: unreadable argon2 version")
	}
	if version != argon2.Version {
		return Argon2Params{}, kernel.Invalidf("auth: unsupported argon2 version %d", version)
	}
	var p Argon2Params
	if _, err := fmt.Sscanf(paramPart, "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return Argon2Params{}, kernel.Invalid("auth: unreadable argon2 parameters")
	}
	if err := validateArgon2Params(p); err != nil {
		return Argon2Params{}, err
	}
	return p, nil
}

// validateArgon2Params rejects cost parameters that would crash argon2.IDKey
// (t<1 or p<1), violate argon2's own m>=8*p minimum, or invite resource
// exhaustion (m/t/p above safe ceilings).
func validateArgon2Params(p Argon2Params) error {
	switch {
	case p.Iterations < 1 || p.Parallelism < 1:
		return kernel.Invalid("auth: argon2 iterations and parallelism must be >= 1")
	case p.Memory < 8*uint32(p.Parallelism):
		return kernel.Invalid("auth: argon2 memory below the 8*parallelism minimum")
	case p.Memory > maxArgon2Memory || p.Iterations > maxArgon2Iterations || p.Parallelism > maxArgon2Parallelism:
		return kernel.Invalid("auth: argon2 parameters exceed safe bounds")
	}
	return nil
}

var _ app.PasswordHasher = (*Argon2idHasher)(nil)
