package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

func TestUserEnumMappings(t *testing.T) {
	for _, r := range []identity.Role{identity.RoleEmployer, identity.RoleRecruiter, identity.RoleCandidate} {
		assert.Equal(t, r, userRoleFromDB(userRoleToDB(r)))
	}
	for _, s := range []identity.AccountStatus{identity.StatusActive, identity.StatusLocked} {
		assert.Equal(t, s, userStatusFromDB(userStatusToDB(s)))
	}
}

func TestPassportMapping(t *testing.T) {
	for _, p := range []talent.PassportStatus{talent.PassportCVOnly, talent.PassportScreened, talent.PassportVerified} {
		assert.Equal(t, p, passportFromDB(passportToDB(p)))
	}
}

func TestConfidenceMapping(t *testing.T) {
	for _, c := range []kernel.Confidence{kernel.ConfidenceLow, kernel.ConfidenceMedium, kernel.ConfidenceHigh} {
		assert.Equal(t, c, confidenceFromDB(confidenceToDB(c)))
	}
	assert.Equal(t, kernel.ConfidenceUnknown, confidenceFromDB(confidenceToDB(kernel.ConfidenceUnknown)))
}

func TestApplicationEnumMappings(t *testing.T) {
	for _, s := range []candidateagent.ApplicationSource{candidateagent.SourceManual, candidateagent.SourceAgent} {
		assert.Equal(t, s, appSourceFromDB(appSourceToDB(s)))
	}
	for _, s := range []candidateagent.ApplicationStatus{candidateagent.StatusDrafted, candidateagent.StatusSubmitted, candidateagent.StatusScreening, candidateagent.StatusScreened} {
		assert.Equal(t, s, appStatusFromDB(appStatusToDB(s)))
	}
}
