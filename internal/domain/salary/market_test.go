package salary_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/salary"
)

func TestLookup_ScalesWithSeniority(t *testing.T) {
	junior := salary.Lookup("Backend Engineer", role.SeniorityJunior)
	mid := salary.Lookup("Backend Engineer", role.SeniorityMid)
	senior := salary.Lookup("Backend Engineer", role.SenioritySenior)
	lead := salary.Lookup("Backend Engineer", role.SeniorityLead)

	// Monotonic: each step up pays at least as much as the one below.
	assert.Less(t, junior.High, mid.High)
	assert.Less(t, mid.High, senior.High)
	assert.Less(t, senior.High, lead.High)
	assert.Less(t, junior.Low, mid.Low)
	assert.Less(t, mid.Low, senior.Low)
	assert.Less(t, senior.Low, lead.Low)
}

func TestLookup_AlwaysWellFormedGHS(t *testing.T) {
	for _, s := range []role.Seniority{
		role.SeniorityJunior, role.SeniorityMid, role.SenioritySenior, role.SeniorityLead, role.SeniorityUnspecified,
	} {
		b := salary.Lookup("Software Engineer", s)
		assert.Equal(t, "GHS", b.Currency)
		assert.Positive(t, b.Low)
		assert.GreaterOrEqual(t, b.High, b.Low)
		assert.NoError(t, b.Validate(), "lookup must return a valid band")
	}
}

func TestLookup_UnspecifiedSeniorityIsMidBaseline(t *testing.T) {
	assert.Equal(t,
		salary.Lookup("Backend Engineer", role.SeniorityMid),
		salary.Lookup("Backend Engineer", role.SeniorityUnspecified),
		"an unspecified seniority falls back to the mid-level band")
}

func TestLookup_FamilyPremiumsAndDiscounts(t *testing.T) {
	base := salary.Lookup("Backend Engineer", role.SenioritySenior)
	data := salary.Lookup("Senior Data Scientist", role.SenioritySenior)
	platform := salary.Lookup("Senior DevOps Engineer", role.SenioritySenior)
	design := salary.Lookup("Senior Product Designer", role.SenioritySenior)
	qa := salary.Lookup("Senior QA Engineer", role.SenioritySenior)

	// Data/ML and platform/SRE command a premium over general engineering.
	assert.Greater(t, data.High, base.High)
	assert.Greater(t, platform.High, base.High)
	// Design and QA sit slightly below the engineering baseline.
	assert.Less(t, design.High, base.High)
	assert.Less(t, qa.High, base.High)
}

func TestLookup_RoundsToTidyFigures(t *testing.T) {
	b := salary.Lookup("Senior Data Scientist", role.SenioritySenior)
	assert.Zero(t, int(b.Low)%500, "low rounded to nearest 500")
	assert.Zero(t, int(b.High)%500, "high rounded to nearest 500")
}
