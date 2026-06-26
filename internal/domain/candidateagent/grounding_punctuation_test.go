package candidateagent_test

import (
	"testing"

	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"

	"github.com/stretchr/testify/assert"
)

// TestCheckGrounding_PunctuatedSkillsTokenizeLikeTheGate guards the fix for the
// tokenizer divergence: grounding now tokenizes punctuated skill names exactly
// as the must-have coverage gate does, so the two gates can never disagree.
func TestCheckGrounding_PunctuatedSkillsTokenizeLikeTheGate(t *testing.T) {
	// Under-block fixed: a C++ profile does NOT silently cover a separate "C"
	// claim (previously "C++" was stripped to "c" and matched "C").
	under := agentdom.CheckGrounding("Experienced in C programming.", []string{"Go", "C++"}, []string{"C"})
	assert.False(t, under.Grounded, "C++ does not cover an unverified C claim")
	assert.Equal(t, []string{"C"}, under.Fabricated)

	// Over-block fixed: a verified "C++ / Systems" profile covers an honest "C++"
	// claim (previously "C++ / Systems" was stripped to "c systems" and missed).
	over := agentdom.CheckGrounding("Strong C++ background.", []string{"C++ / Systems"}, []string{"C++"})
	assert.True(t, over.Grounded, "a verified C++ skill is not flagged on an eligible candidate")

	// .NET and C# remain whole tokens, so a matching profile covers them.
	dotnet := agentdom.CheckGrounding("Built services in .NET and C#.", []string{".NET", "C#"}, []string{".NET", "C#"})
	assert.True(t, dotnet.Grounded)
}
