package matching_test

import (
	"testing"

	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"

	"github.com/stretchr/testify/assert"
)

func TestViolatesDealBreaker(t *testing.T) {
	assert.False(t, matchingdom.ViolatesDealBreaker(nil, "Requires on-call rotation"), "no deal-breakers never excludes")
	assert.True(t, matchingdom.ViolatesDealBreaker([]string{"on-call"}, "Participate in on-call rotation."),
		"single-token deal-breaker present in the role text")
	assert.True(t, matchingdom.ViolatesDealBreaker([]string{"night shift"}, "Occasional night shift work expected."),
		"multi-word deal-breaker matched as a contiguous phrase")
	assert.False(t, matchingdom.ViolatesDealBreaker([]string{"relocation"}, "Fully remote, no travel."),
		"deal-breaker absent from the role text")
	assert.False(t, matchingdom.ViolatesDealBreaker([]string{"call"}, "An on-call-free culture."),
		"whole-token: 'call' is not found inside 'on-call-free'")
}
