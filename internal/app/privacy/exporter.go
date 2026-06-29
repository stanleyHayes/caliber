// Package privacy holds data-subject-rights use-cases (CAL-118, Ghana DPA 2012).
// The right of access (DSAR) is implemented here: a candidate can obtain a
// complete, structured copy of every record the platform holds about them. The
// matching erasure path is designed in docs/data-protection.md and lands next.
package privacy

import (
	"context"

	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	contestdom "github.com/xcreativs/caliber/internal/domain/contest"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// exportPageSize bounds each repository read while paging through a subject's
// full history; the exporter keeps reading until every record is collected, so
// this is a batch size, never a cap on the result.
const exportPageSize = 100

// DataExport is the complete copy of a candidate's data returned for a DSAR
// (Ghana DPA 2012, right of access). Every record the platform holds about the
// subject is included so they can see exactly what is processed about them.
type DataExport struct {
	Candidate    *talent.Candidate
	Profile      *talent.TalentProfile // nil if the candidate never built a passport
	Applications []*agentdom.Application
	Interviews   []*interviewdom.Interview
	Contests     []*contestdom.Contest
}

// Exporter assembles a candidate's full data export from the read side of the
// domain ports — it never mutates anything.
type Exporter struct {
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	apps       agentdom.ApplicationRepository
	interviews interviewdom.InterviewRepository
	contests   contestdom.ContestRepository
}

// NewExporter wires the DSAR use-case over the repositories it reads.
func NewExporter(
	candidates talent.CandidateRepository,
	profiles talent.TalentProfileRepository,
	apps agentdom.ApplicationRepository,
	interviews interviewdom.InterviewRepository,
	contests contestdom.ContestRepository,
) *Exporter {
	return &Exporter{candidates: candidates, profiles: profiles, apps: apps, interviews: interviews, contests: contests}
}

// ExportCandidate gathers every record held about a candidate. The candidate
// must exist; a never-built profile is simply omitted (not an error). The
// caller is responsible for authorizing the subject (candidate-self).
func (e *Exporter) ExportCandidate(ctx context.Context, candidateID kernel.ID) (*DataExport, error) {
	cand, err := e.candidates.ByID(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	out := &DataExport{Candidate: cand}

	// A candidate may never have built a passport — that is a valid empty export,
	// not a failure. Any other error is surfaced.
	profile, perr := e.profiles.ByCandidateID(ctx, candidateID)
	switch {
	case perr == nil:
		out.Profile = profile
	case kernel.KindOf(perr) != kernel.KindNotFound:
		return nil, perr
	}

	apps, err := collectAll(func(p kernel.Page) ([]*agentdom.Application, int64, error) {
		return e.apps.ByCandidate(ctx, candidateID, p)
	})
	if err != nil {
		return nil, err
	}
	out.Applications = apps

	interviews, err := collectAll(func(p kernel.Page) ([]*interviewdom.Interview, int64, error) {
		return e.interviews.ByCandidate(ctx, candidateID, p)
	})
	if err != nil {
		return nil, err
	}
	out.Interviews = interviews

	contests, err := collectAll(func(p kernel.Page) ([]*contestdom.Contest, int64, error) {
		return e.contests.ByCandidate(ctx, candidateID, p)
	})
	if err != nil {
		return nil, err
	}
	out.Contests = contests

	return out, nil
}

// collectAll pages through a paginated repository read until every record is
// gathered, so a DSAR export is complete and never silently truncated. It is
// bounded by the reported total, so a repo that misreports total cannot loop
// forever.
func collectAll[T any](read func(kernel.Page) ([]*T, int64, error)) ([]*T, error) {
	var all []*T
	for page := 1; ; page++ {
		batch, total, err := read(kernel.NewPage(page, exportPageSize))
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) == 0 || int64(len(all)) >= total {
			return all, nil
		}
	}
}
