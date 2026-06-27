package candidateagent_test

import (
	"testing"

	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"

	"github.com/stretchr/testify/assert"
)

// TestCheckGrounding_SynonymEvasionIsCaught closes the bypass the review found:
// claiming an uncovered role competency by a common abbreviation must NOT slip
// past the guard.
func TestCheckGrounding_SynonymEvasionIsCaught(t *testing.T) {
	g := agentdom.CheckGrounding("I run k8s clusters in production.", []string{"Go"}, []string{"Kubernetes"})
	assert.False(t, g.Grounded, "k8s canonicalizes to Kubernetes and is flagged when the profile lacks it")
	assert.Equal(t, []string{"Kubernetes"}, g.Fabricated)

	g2 := agentdom.CheckGrounding("Golang expert.", []string{"Python"}, []string{"Go"})
	assert.False(t, g2.Grounded, "golang canonicalizes to Go")
}

// TestCheckGrounding_SynonymCoverageAvoidsFalsePositive is the mirror: an honest
// claim written as a variant of a skill the profile genuinely has must not be
// flagged.
func TestCheckGrounding_SynonymCoverageAvoidsFalsePositive(t *testing.T) {
	// Profile carries "k8s"; the role asks for "Kubernetes"; the honest claim clears.
	g := agentdom.CheckGrounding("Strong Kubernetes background.", []string{"k8s"}, []string{"Kubernetes"})
	assert.True(t, g.Grounded, "a k8s profile covers a Kubernetes role")

	// Profile "PostgreSQL" covers a "Postgres" role competency.
	g2 := agentdom.CheckGrounding("Deep Postgres experience.", []string{"PostgreSQL"}, []string{"Postgres"})
	assert.True(t, g2.Grounded)
}
