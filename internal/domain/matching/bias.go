package matching

import (
	"sort"
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// protectedAttributes is the canonical, lower-cased set of attributes that
// MUST never influence scoring or ranking. It is kept private so callers go
// through ProtectedAttributes (read-only copy) and EnsureBiasSafe (validation).
var protectedAttributes = map[string]struct{}{ //nolint:gochecknoglobals // canonical immutable protected-attribute set
	"gender":         {},
	"age":            {},
	"ethnicity":      {},
	"religion":       {},
	"nationality":    {},
	"marital_status": {},
	"disability":     {},
}

// ProtectedAttributes returns a sorted copy of the protected attribute keys
// that must never be used as ranking or scoring signals.
func ProtectedAttributes() []string {
	out := make([]string, 0, len(protectedAttributes))
	for k := range protectedAttributes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// EnsureBiasSafe verifies that no signal key is a protected attribute. The
// comparison is case-insensitive and ignores surrounding whitespace. It
// returns a kernel.Invalid error naming the first offending key. Ranking
// inputs MUST pass EnsureBiasSafe before any scoring is performed.
func EnsureBiasSafe(signalKeys []string) error {
	for _, key := range signalKeys {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if _, bad := protectedAttributes[normalized]; bad {
			return kernel.Invalidf("matching: signal key %q is a protected attribute and must not be used for ranking", key)
		}
	}
	return nil
}
