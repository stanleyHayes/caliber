package logging

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

// redactedPlaceholder replaces any value scrubbed from a log line.
const redactedPlaceholder = "[REDACTED]"

// redactingHandler wraps an slog.Handler and removes personal data from every
// record before it is emitted (CAL-117). It is a defense-in-depth net, not a
// licence to log PII: call sites still avoid it deliberately (data-protection.md).
// Two passes run on every attribute (recursively through groups) and on the
// message: a sensitive-key denylist blanks values whose key names a secret or an
// identifier, and a value scan masks PII-shaped substrings (emails, bearer
// tokens, JWTs) wherever they appear — even inside an otherwise innocuous field.
type redactingHandler struct {
	inner slog.Handler
}

// newRedactingHandler wraps inner so all output is scrubbed.
func newRedactingHandler(inner slog.Handler) slog.Handler {
	return redactingHandler{inner: inner}
}

// Patterns for PII-shaped substrings. They are intentionally broad: over-masking
// a log line is harmless, leaking a credential or an email is not.
var (
	emailPattern = regexp.MustCompile(`[\w.+-]+@[\w-]+\.[\w.-]+`)
	// A bearer credential as it might be mistakenly logged from a header.
	bearerPattern = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/-]+=*`)
	// A JWT (three base64url segments) — access/refresh tokens.
	jwtPattern = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
)

// isSensitiveKey reports whether an attribute key names a secret or a personal
// identifier, so its value is always blanked. The exact set covers credential
// and PII field names; the substring pass then catches variants like user_email
// or db_password — but only for strong markers, so neutral keys such as
// token_count or service_name (which a bare "token"/"name" match would catch)
// are left intact.
func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	switch k {
	case "email", "e_mail", "mail",
		"password", "passwd", "pwd",
		"secret", "authorization", "auth_header",
		"token", "access_token", "refresh_token",
		"api_key", "apikey", "cookie", "set_cookie",
		"phone", "telephone", "msisdn",
		"ssn", "dob", "date_of_birth":
		return true
	}
	for _, marker := range [...]string{"password", "secret", "email"} {
		if strings.Contains(k, marker) {
			return true
		}
	}
	return false
}

// redactString masks every PII-shaped substring in s.
func redactString(s string) string {
	s = emailPattern.ReplaceAllString(s, redactedPlaceholder)
	s = bearerPattern.ReplaceAllString(s, redactedPlaceholder)
	s = jwtPattern.ReplaceAllString(s, redactedPlaceholder)
	return s
}

// redactAttr returns a copy of a with its value scrubbed, recursing into groups.
func redactAttr(a slog.Attr) slog.Attr {
	if isSensitiveKey(a.Key) {
		return slog.String(a.Key, redactedPlaceholder)
	}
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindGroup:
		group := v.Group()
		out := make([]any, 0, len(group))
		for _, ga := range group {
			out = append(out, redactAttr(ga))
		}
		return slog.Group(a.Key, out...)
	case slog.KindString:
		return slog.String(a.Key, redactString(v.String()))
	default:
		return slog.Attr{Key: a.Key, Value: v}
	}
}

func (h redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h redactingHandler) Handle(ctx context.Context, r slog.Record) error {
	clone := slog.NewRecord(r.Time, r.Level, redactString(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clone.AddAttrs(redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, clone)
}

func (h redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	scrubbed := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		scrubbed[i] = redactAttr(a)
	}
	return redactingHandler{inner: h.inner.WithAttrs(scrubbed)}
}

func (h redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{inner: h.inner.WithGroup(name)}
}
