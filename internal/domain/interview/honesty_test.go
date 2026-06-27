package interview_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
)

func TestVagueAnswer_FlagsThinOrEvasiveAnswers(t *testing.T) {
	vague := []string{
		"",
		"It was good.",
		"I basically just did various things to help out.",
		"We sort of handled it as a team, you know.",
		"Generally I work on whatever needs doing.",
		"Yeah, lots of stuff with the backend and things.",
	}
	for _, a := range vague {
		assert.Truef(t, interviewdom.VagueAnswer(a), "expected vague: %q", a)
	}
}

func TestVagueAnswer_AcceptsConcreteAnswers(t *testing.T) {
	concrete := []string{
		"I led the migration of our payments service to Go, which cut p99 latency by 40%.",
		"I built the rate limiter using Redis and it handled 3000 requests per second.",
		"My role was to design the schema; I reduced the nightly batch from 6 hours to 45 minutes.",
		"I personally rewrote the auth flow and shipped it to 12000 users.",
	}
	for _, a := range concrete {
		assert.Falsef(t, interviewdom.VagueAnswer(a), "expected concrete: %q", a)
	}
}

func TestVagueAnswer_DigitCountsAsConcreteSignal(t *testing.T) {
	// A number is concrete anchor enough to avoid the vague flag.
	assert.False(t, interviewdom.VagueAnswer("We cut the error rate to 0.2% over 3 weeks."))
}

func TestVagueAnswer_LongSpecificAnswerWithoutHedgesPasses(t *testing.T) {
	// No digit, no first-person ownership phrase, but a long, specific, non-hedging
	// answer gets the benefit of the doubt (not flagged).
	a := "The caching layer sat in front of the catalogue service and served read traffic " +
		"while writes went straight through to the primary database for consistency."
	assert.False(t, interviewdom.VagueAnswer(a))
}
