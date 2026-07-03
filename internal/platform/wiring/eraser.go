package wiring

import (
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
	"github.com/xcreativs/caliber/internal/domain/audit"
)

// BuildEraser wires the right-to-erasure cascade (CAL-118) from whichever
// repositories are configured. Each dependency is taken through its privacy port
// rather than a concrete type, so the same wiring serves both the in-memory dev
// stack and the Postgres stack — candidate/profile/application/match/user provide
// the hard-delete primitives on both, while interviews, contests, and the audit
// trail remain in-memory on both paths. It returns nil only if some repository
// lacks an erasure primitive, so callers can degrade (report unimplemented, skip
// the retention sweep) rather than panic.
func BuildEraser(repos Repositories, auditRepo audit.AuditRepository, contests *memory.ContestRepo) *privacyapp.Eraser {
	candidate, ok1 := repos.Candidates.(privacyapp.CandidateRemover)
	identity, ok2 := repos.Users.(privacyapp.IdentityAnonymizer)
	aud, ok3 := auditRepo.(privacyapp.AuditTombstoner)
	profs, ok4 := repos.Profiles.(privacyapp.CandidateScopedEraser)
	apps, ok5 := repos.Apps.(privacyapp.CandidateScopedEraser)
	ivs, ok6 := repos.Interviews.(privacyapp.CandidateScopedEraser)
	matches, ok7 := repos.Matches.(privacyapp.CandidateScopedEraser)
	if ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7 {
		return privacyapp.NewEraser(candidate, identity, aud, profs, apps, ivs, matches, contests)
	}
	return nil
}
