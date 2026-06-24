package identity

import (
	"regexp"
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// emailRe is a pragmatic email format check (not full RFC 5322).
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Email is a validated, normalized email address.
type Email string

// NewEmail validates and normalizes (trim + lowercase) an email address.
func NewEmail(raw string) (Email, error) {
	e := strings.ToLower(strings.TrimSpace(raw))
	if !emailRe.MatchString(e) {
		return "", kernel.Invalidf("identity: invalid email %q", raw)
	}
	return Email(e), nil
}

// String returns the normalized email value.
func (e Email) String() string { return string(e) }
