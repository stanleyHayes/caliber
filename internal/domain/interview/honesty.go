package interview

import (
	"strings"
	"unicode"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// concreteTokenFloor is the answer length (in tokens) below which an answer with
// no concrete anchor reads as too thin to count as honest signal.
const concreteTokenFloor = 12

// VagueAnswer reports whether a candidate's interview answer reads as vague or
// evasive — i.e. it lacks the concrete signal a screening needs: a specific
// example, the candidate's own actions, or measurable detail.
//
// It powers honest-signal pressure (CAL-063): when true, the interviewer presses
// for a concrete example instead of accepting the answer and moving on. The
// heuristic is intentionally lenient toward catching vagueness — a false positive
// just means one extra "give me a specific example" follow-up, which is harmless
// in a screening, whereas a false negative lets an evasive answer pass unchallenged.
//
// Scope: this is a surface heuristic over the answer text only; it does not judge
// truthfulness (that is the scorer's job over the whole transcript). An answer
// that names a concrete first-person action or includes a number is treated as
// having signal and is never flagged.
func VagueAnswer(answer string) bool {
	lower := strings.ToLower(answer)
	tokens := kernel.Tokens(lower)
	if hasConcreteSignal(lower) {
		return false
	}
	// No concrete anchor: vague when the answer is short, or hedges at all.
	return len(tokens) < concreteTokenFloor || hedgeCount(lower) >= 1
}

// hasConcreteSignal reports whether the (lower-cased) answer contains a concrete
// anchor: a digit (a count, percentage, duration) or a first-person ownership
// phrase that ties the candidate to a specific action.
func hasConcreteSignal(lower string) bool {
	if strings.ContainsFunc(lower, unicode.IsDigit) {
		return true
	}
	ownership := []string{
		"i led", "i built", "i designed", "i shipped", "i implemented", "i developed",
		"i wrote", "i created", "i migrated", "i architected", "i owned", "i reduced",
		"i increased", "i improved", "i delivered", "i debugged", "i refactored",
		"my role was", "i was responsible", "i personally", "i decided",
	}
	for _, phrase := range ownership {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// hedgeCount counts hedging / filler markers in the (lower-cased) answer — the
// language of an answer that gestures at competence without committing to specifics.
func hedgeCount(lower string) int {
	markers := []string{
		"basically", "generally", "various", "stuff", "kind of", "sort of",
		"a lot of", "you know", "i guess", "probably", "somewhat", "more or less",
		"and things", "or whatever", "et cetera", "lots of things", "all sorts",
	}
	count := 0
	for _, m := range markers {
		if strings.Contains(lower, m) {
			count++
		}
	}
	return count
}
