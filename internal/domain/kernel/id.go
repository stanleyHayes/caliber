// Package kernel holds the shared domain primitives every bounded context
// builds on: identifiers, value objects, pagination, and the typed error model.
// It is pure (standard library only) and imports nothing from the rest of the app.
package kernel

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

// ID is an opaque entity identifier (hex-encoded random bytes).
type ID string

// NewID returns a new random identifier.
func NewID() ID {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return ID(hex.EncodeToString(b[:]))
}

// String returns the raw identifier value.
func (id ID) String() string { return string(id) }

// IsZero reports whether the identifier is empty.
func (id ID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// IDFromString validates and converts a raw string to an ID.
func IDFromString(s string) (ID, error) {
	if strings.TrimSpace(s) == "" {
		return "", errors.New("kernel: id cannot be empty")
	}
	return ID(s), nil
}
