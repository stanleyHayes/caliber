package privacy_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/app/privacy"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

func TestDataExportJSON_IsCompleteAndStable(t *testing.T) {
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(cid, "Accra", talent.CandidateIntake{Location: "Accra"})
	require.NoError(t, err)
	prof, err := talent.NewTalentProfile(cid, "Backend engineer.", []talent.ProfileCompetency{
		{Name: "Go", Level: 4, EvidenceQuote: "built services", SourceSpan: "CV"},
	})
	require.NoError(t, err)

	export := &privacy.DataExport{
		Candidate:    cand,
		Profile:      prof,
		Applications: []*agentdom.Application{{ID: kernel.NewID(), CandidateID: cid}},
	}
	b, err := export.JSON()
	require.NoError(t, err)

	// Valid JSON with every top-level section present.
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &doc))
	for _, section := range []string{"candidate", "profile", "applications", "interviews", "contests"} {
		assert.Contains(t, doc, section, "the export always includes the %q section", section)
	}
	// Empty sections serialise as [] (not null), so the shape is predictable.
	assert.JSONEq(t, `[]`, string(doc["interviews"]))
	assert.JSONEq(t, `[]`, string(doc["contests"]))
	// The candidate's data is actually present.
	assert.Contains(t, string(b), "Backend engineer.")
}

func TestDataExportJSON_AbsentProfileIsNull(t *testing.T) {
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	export := &privacy.DataExport{Candidate: cand} // no profile

	b, err := export.JSON()
	require.NoError(t, err)
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &doc))
	assert.JSONEq(t, `null`, string(doc["profile"]), "a never-built profile is explicit null")
}
