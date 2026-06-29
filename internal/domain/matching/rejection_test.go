package matching_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/matching"
)

func TestNewRejection_RejectsOversizedReason(t *testing.T) {
	// Untrusted human input is length-capped at the domain boundary (CAL-111).
	huge := strings.Repeat("a", matching.MaxReasonLen+1)
	_, err := matching.NewRejection(kernel.NewID(), kernel.NewID(), huge, true)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestNewRejection_RequiresHumanApproval(t *testing.T) {
	// The defining invariant (CAL-081): the system never auto-rejects. A
	// rejection cannot be constructed without an explicit human affirmation,
	// even when every other field is valid.
	_, err := matching.NewRejection(kernel.NewID(), kernel.NewID(), "not a fit for the seniority bar", false)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestNewRejection_RequiresReason(t *testing.T) {
	// A decline must be explainable: an empty or whitespace-only reason is rejected.
	for _, reason := range []string{"", "   ", "\t\n"} {
		_, err := matching.NewRejection(kernel.NewID(), kernel.NewID(), reason, true)
		require.Errorf(t, err, "reason %q should be rejected", reason)
		assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
	}
}

func TestNewRejection_RequiresRoleAndCandidate(t *testing.T) {
	_, err := matching.NewRejection("", kernel.NewID(), "reason", true)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))

	_, err = matching.NewRejection(kernel.NewID(), "", "reason", true)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestNewRejection_SanitisesAndTrimsReason(t *testing.T) {
	roleID, candidateID := kernel.NewID(), kernel.NewID()
	rej, err := matching.NewRejection(roleID, candidateID, "  Strong engineer, but the role needs on-site presence  ", true)
	require.NoError(t, err)
	assert.Equal(t, roleID, rej.RoleID)
	assert.Equal(t, candidateID, rej.CandidateID)
	assert.Equal(t, "Strong engineer, but the role needs on-site presence", rej.Reason)
}

func TestRejection_SnapshotJSON(t *testing.T) {
	roleID, candidateID := kernel.NewID(), kernel.NewID()
	rej, err := matching.NewRejection(roleID, candidateID, "below the must-have Go bar", true)
	require.NoError(t, err)

	snap, err := rej.SnapshotJSON()
	require.NoError(t, err)

	var decoded struct {
		RoleID        string `json:"role_id"`
		CandidateID   string `json:"candidate_id"`
		Reason        string `json:"reason"`
		HumanApproved bool   `json:"human_approved"`
	}
	require.NoError(t, json.Unmarshal([]byte(snap), &decoded))
	assert.Equal(t, roleID.String(), decoded.RoleID)
	assert.Equal(t, candidateID.String(), decoded.CandidateID)
	assert.Equal(t, "below the must-have Go bar", decoded.Reason)
	assert.True(t, decoded.HumanApproved, "a constructed rejection always records the human approval")
}
