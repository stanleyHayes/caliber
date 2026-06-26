package candidateagent_test

import (
	"testing"

	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"

	"github.com/stretchr/testify/assert"
)

func TestCheckGrounding_GroundedWhenOnlyVerifiedSkillsClaimed(t *testing.T) {
	profile := []string{"Go", "SQL"}
	role := []string{"Go", "SQL", "Kubernetes"}
	// Summary mentions only verified skills (Go, SQL), not the unverified Kubernetes.
	g := agentdom.CheckGrounding("Drawing on verified experience in Go and SQL, a strong fit.", profile, role)
	assert.True(t, g.Grounded)
	assert.Empty(t, g.Fabricated)
}

func TestCheckGrounding_FlagsFabricatedRoleSkill(t *testing.T) {
	profile := []string{"Go", "SQL"}
	role := []string{"Go", "SQL", "Kubernetes"}
	// The summary claims Kubernetes, which the profile does not evidence.
	g := agentdom.CheckGrounding("Experienced in Go, SQL, and Kubernetes orchestration.", profile, role)
	assert.False(t, g.Grounded)
	assert.Equal(t, []string{"Kubernetes"}, g.Fabricated)
}

func TestCheckGrounding_TokenCoverageMatchesMustHaveGate(t *testing.T) {
	// Profile "SQL / Databases" covers the role's "SQL" must-have, so claiming
	// SQL is grounded.
	profile := []string{"Go", "SQL / Databases"}
	role := []string{"SQL"}
	g := agentdom.CheckGrounding("Strong SQL background.", profile, role)
	assert.True(t, g.Grounded)
}

func TestCheckGrounding_WholeTokenNotSubstring(t *testing.T) {
	profile := []string{"Python"}
	role := []string{"Go"}
	// "ago"/"going" must not count as claiming "Go".
	g := agentdom.CheckGrounding("A while ago I was going to learn more.", profile, role)
	assert.True(t, g.Grounded, "substring 'go' inside other words is not a Go claim")

	g2 := agentdom.CheckGrounding("I write Go daily.", profile, role)
	assert.False(t, g2.Grounded, "the standalone token 'Go' is an unverified claim")
}

func TestCheckGrounding_MultiWordCompetency(t *testing.T) {
	profile := []string{"Go"}
	role := []string{"System design"}
	g := agentdom.CheckGrounding("Led complex system design across services.", profile, role)
	assert.False(t, g.Grounded)
	assert.Equal(t, []string{"System design"}, g.Fabricated)

	// The same words out of order do not assert the competency phrase.
	g2 := agentdom.CheckGrounding("Designed a flexible system of records.", profile, role)
	assert.True(t, g2.Grounded, "non-contiguous tokens are not the competency phrase")
}

func TestCheckGrounding_EmptyInputsAreVacuouslyGrounded(t *testing.T) {
	assert.True(t, agentdom.CheckGrounding("", []string{"Go"}, []string{"Go"}).Grounded)
	assert.True(t, agentdom.CheckGrounding("anything", []string{"Go"}, nil).Grounded)
	assert.True(t, agentdom.CheckGrounding("anything", nil, []string{"  "}).Grounded, "blank role competency is skipped")
}

func TestCheckGrounding_CaseInsensitive(t *testing.T) {
	profile := []string{"go"}
	role := []string{"GO", "kubernetes"}
	g := agentdom.CheckGrounding("Built services in GO; also ran KUBERNETES clusters.", profile, role)
	assert.False(t, g.Grounded)
	assert.Equal(t, []string{"kubernetes"}, g.Fabricated, "GO is covered (case-insensitive); kubernetes is not")
}
