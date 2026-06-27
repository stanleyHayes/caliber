package matching

import (
	"encoding/json"
	"strings"

	"github.com/xcreativs/caliber/internal/domain/guard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Rejection is a human's approved decision to decline a candidate for a role.
//
// CAL-081 — the system never auto-rejects. The AI ranks and screens, but a
// rejection comes into being only as a deliberate, explainable human act: an
// explicit approval accompanied by a reason. The constructor is the gate; there
// is no path that produces a Rejection without that affirmation.
type Rejection struct {
	RoleID      kernel.ID
	CandidateID kernel.ID
	Reason      string
}

// NewRejection enforces the human-approval invariant and returns a Rejection.
//
// humanApproved must be true — there is no automated rejection path — and a
// non-empty reason is required so that every decline is explainable. The reason
// is sanitised (it is human-entered, untrusted text) before being retained.
func NewRejection(roleID, candidateID kernel.ID, reason string, humanApproved bool) (Rejection, error) {
	if !humanApproved {
		return Rejection{}, kernel.Invalid("rejection: a human must approve every decline; the system never auto-rejects")
	}
	if roleID.IsZero() || candidateID.IsZero() {
		return Rejection{}, kernel.Invalid("rejection: role and candidate are required")
	}
	reason = strings.TrimSpace(guard.Sanitize(reason))
	if reason == "" {
		return Rejection{}, kernel.Invalid("rejection: a reason is required (a decline must be explainable)")
	}
	return Rejection{RoleID: roleID, CandidateID: candidateID, Reason: reason}, nil
}

// SnapshotJSON renders the audited after-state of the rejection. human_approved
// is always true: a Rejection cannot be constructed otherwise, so the snapshot
// records the affirmation that made the decline legitimate.
func (r Rejection) SnapshotJSON() (string, error) {
	b, err := json.Marshal(struct {
		RoleID        string `json:"role_id"`
		CandidateID   string `json:"candidate_id"`
		Reason        string `json:"reason"`
		HumanApproved bool   `json:"human_approved"`
	}{
		RoleID:        r.RoleID.String(),
		CandidateID:   r.CandidateID.String(),
		Reason:        r.Reason,
		HumanApproved: true,
	})
	if err != nil {
		return "", kernel.Wrap(err, kernel.KindInternal, "rejection: encode snapshot")
	}
	return string(b), nil
}
