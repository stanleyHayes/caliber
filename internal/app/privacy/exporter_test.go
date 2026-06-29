package privacy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app/privacy"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	contestdom "github.com/xcreativs/caliber/internal/domain/contest"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"
)

type deps struct {
	candidates *mocks.MockCandidateRepository
	profiles   *mocks.MockTalentProfileRepository
	apps       *mocks.MockApplicationRepository
	interviews *mocks.MockInterviewRepository
	contests   *mocks.MockContestRepository
}

func newDeps(ctrl *gomock.Controller) deps {
	return deps{
		candidates: mocks.NewMockCandidateRepository(ctrl),
		profiles:   mocks.NewMockTalentProfileRepository(ctrl),
		apps:       mocks.NewMockApplicationRepository(ctrl),
		interviews: mocks.NewMockInterviewRepository(ctrl),
		contests:   mocks.NewMockContestRepository(ctrl),
	}
}

func (d deps) exporter() *privacy.Exporter {
	return privacy.NewExporter(d.candidates, d.profiles, d.apps, d.interviews, d.contests)
}

func TestExportCandidate_GathersEverything(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()

	cand, err := talent.NewCandidate(cid, "Accra", talent.CandidateIntake{Location: "Accra"})
	require.NoError(t, err)
	prof, err := talent.NewTalentProfile(cid, "summary", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	iv, err := interviewdom.NewInterview(kernel.NewID(), cid, interviewdom.ModeText)
	require.NoError(t, err)
	con, err := contestdom.NewContest(cid, kernel.NewID(), contestdom.SubjectReportCard, "missed my Go work", time.Unix(1, 0))
	require.NoError(t, err)
	app := &agentdom.Application{ID: kernel.NewID(), CandidateID: cid}

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(prof, nil)
	d.apps.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return([]*agentdom.Application{app}, int64(1), nil)
	d.interviews.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return([]*interviewdom.Interview{iv}, int64(1), nil)
	d.contests.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return([]*contestdom.Contest{con}, int64(1), nil)

	out, err := d.exporter().ExportCandidate(context.Background(), cid)
	require.NoError(t, err)
	assert.Equal(t, cand, out.Candidate)
	assert.Equal(t, prof, out.Profile)
	require.Len(t, out.Applications, 1)
	require.Len(t, out.Interviews, 1)
	require.Len(t, out.Contests, 1)
	assert.Equal(t, app.ID, out.Applications[0].ID)
}

func TestExportCandidate_OmitsAbsentProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(cid, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	// A candidate who never built a passport: NotFound is a valid empty export.
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	d.apps.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)
	d.interviews.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)
	d.contests.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)

	out, err := d.exporter().ExportCandidate(context.Background(), cid)
	require.NoError(t, err)
	assert.Nil(t, out.Profile, "a never-built passport is omitted, not an error")
	assert.Empty(t, out.Applications)
	assert.Empty(t, out.Interviews)
	assert.Empty(t, out.Contests)
}

func TestExportCandidate_UnknownCandidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.candidates.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	_, err := d.exporter().ExportCandidate(context.Background(), kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestExportCandidate_PagesThroughEveryRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(cid, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)

	// 250 applications span three batch reads (100 + 100 + 50); the export must
	// collect all of them, never silently truncating at one page.
	const totalApps = 250
	backing := make([]*agentdom.Application, totalApps)
	for i := range backing {
		backing[i] = &agentdom.Application{ID: kernel.NewID(), CandidateID: cid}
	}

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	d.apps.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ kernel.ID, p kernel.Page) ([]*agentdom.Application, int64, error) {
			start := p.Offset()
			end := min(start+p.Limit(), totalApps)
			if start >= totalApps {
				return nil, int64(totalApps), nil
			}
			return backing[start:end], int64(totalApps), nil
		}).AnyTimes()
	d.interviews.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)
	d.contests.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), nil)

	out, err := d.exporter().ExportCandidate(context.Background(), cid)
	require.NoError(t, err)
	assert.Len(t, out.Applications, totalApps, "every page is collected")
}

func TestExportCandidate_PropagatesReadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(cid, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	d.apps.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(nil, int64(0), errors.New("db down"))

	_, err = d.exporter().ExportCandidate(context.Background(), cid)
	require.Error(t, err)
}
