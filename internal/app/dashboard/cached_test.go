package dashboard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// manualClock is a test clock with explicit advancement.
type manualClock struct {
	t time.Time
}

func newManualClock(t time.Time) *manualClock { return &manualClock{t: t} }

func (m *manualClock) Now() time.Time { return m.t }

func (m *manualClock) Advance(d time.Duration) { m.t = m.t.Add(d) }

func TestCachedPool_HitsCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cand, user, profile := poolFixtures(t)

	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*talent.Candidate{cand}, int64(1), nil).Times(1)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil).Times(1)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(profile, nil).Times(1)

	cached := dashboardapp.NewCachedAggregator(d.agg(), time.Minute, nil)
	page := kernel.NewPage(1, 10)

	first, total1, err := cached.Pool(context.Background(), page)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total1)
	assert.Equal(t, "Ama Mensah", first[0].Name)

	second, total2, err := cached.Pool(context.Background(), page)
	require.NoError(t, err)
	assert.Equal(t, total1, total2)
	assert.Equal(t, first[0].CandidateID, second[0].CandidateID)
}

func TestCachedSupplyDemand_HitsCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := openRole(t, role.SeniorityMid)

	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil).Times(1)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(3), nil).Times(1)

	cached := dashboardapp.NewCachedAggregator(d.agg(), time.Minute, nil)

	first, err := cached.SupplyDemand(context.Background())
	require.NoError(t, err)
	require.Len(t, first, 1)

	second, err := cached.SupplyDemand(context.Background())
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestCachedAlerts_HitsCacheAndPagesInMemory(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cand, user := alertCandidate(t)
	fitRole := alertRole(t, "Backend Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}})

	// The cached aggregator fetches the full alert window once, then pages in memory.
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{fitRole}, int64(1), nil).Times(1)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*talent.Candidate{cand}, int64(1), nil).Times(1)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(goSQLProfileFor(t, cand.ID), nil).Times(1)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil).Times(1)

	cached := dashboardapp.NewCachedAggregator(d.agg(), time.Minute, nil)

	page1, total1, err := cached.Alerts(context.Background(), kernel.NewPage(1, 1))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total1)
	require.Len(t, page1, 1)

	page2, total2, err := cached.Alerts(context.Background(), kernel.NewPage(2, 1))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total2)
	require.Len(t, page2, 1)
	assert.NotEqual(t, page1[0].ID, page2[0].ID, "page 1 and page 2 come from the cached full list")
}

func TestCachedTimeToShortlist_HitsCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cached := dashboardapp.NewCachedAggregator(d.agg(), time.Minute, nil)

	first := cached.TimeToShortlist(context.Background())
	second := cached.TimeToShortlist(context.Background())
	assert.Equal(t, first, second)
}

func TestCachedRefreshSnapshots_InvalidatesCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := openRole(t, role.SeniorityMid)

	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil).Times(2)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(3), nil).Times(2)

	cached := dashboardapp.NewCachedAggregator(d.agg(), time.Minute, nil)

	_, err := cached.SupplyDemand(context.Background())
	require.NoError(t, err)
	cached.RefreshSnapshots()
	_, err = cached.SupplyDemand(context.Background())
	require.NoError(t, err)
}

func TestCachedTTL_ExpiresSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := openRole(t, role.SeniorityMid)

	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil).Times(2)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(3), nil).Times(2)

	ttl := 5 * time.Second
	clock := newManualClock(time.Unix(0, 0))

	cached := dashboardapp.NewCachedAggregator(d.agg(), ttl, clock.Now)

	_, err := cached.SupplyDemand(context.Background())
	require.NoError(t, err)
	clock.Advance(ttl + time.Nanosecond) // push past the snapshot expiry
	_, err = cached.SupplyDemand(context.Background())
	require.NoError(t, err)
}

func TestCachedPool_ImprovesLatency(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cand, user, profile := poolFixtures(t)

	const delay = 50 * time.Millisecond
	// Simulate a slow repository call on the first read only.
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, kernel.Page) ([]*talent.Candidate, int64, error) {
			time.Sleep(delay)
			return []*talent.Candidate{cand}, int64(1), nil
		}).Times(1)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil).Times(1)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(profile, nil).Times(1)

	cached := dashboardapp.NewCachedAggregator(d.agg(), time.Minute, nil)
	page := kernel.NewPage(1, 10)

	first, _, err := cached.Pool(context.Background(), page)
	require.NoError(t, err)

	start := time.Now()
	second, _, err := cached.Pool(context.Background(), page)
	require.NoError(t, err)
	cachedElapsed := time.Since(start)

	assert.Equal(t, first[0].CandidateID, second[0].CandidateID)
	assert.Less(t, cachedElapsed, delay/5, "cached read should be much faster than the slow repository call")
}

func poolFixtures(t *testing.T) (*talent.Candidate, *identity.User, *talent.TalentProfile) {
	t.Helper()
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	email, err := identity.NewEmail("ama@example.com")
	require.NoError(t, err)
	user, err := identity.NewUser(email, identity.RoleCandidate, "Ama Mensah", "hash", time.Unix(1, 0))
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cand.ID, "s",
		[]talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	return cand, user, profile
}
