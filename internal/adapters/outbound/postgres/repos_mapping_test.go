package postgres

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
	"github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

func TestUserEnumMappings(t *testing.T) {
	// RoleAdmin (CAL-154) must round-trip too: without its mapping cases a
	// persisted admin degrades to "unspecified", silently stripping the role.
	for _, r := range []identity.Role{identity.RoleEmployer, identity.RoleRecruiter, identity.RoleCandidate, identity.RoleAdmin} {
		assert.Equal(t, r, userRoleFromDB(userRoleToDB(r)))
	}
	for _, s := range []identity.AccountStatus{identity.StatusActive, identity.StatusLocked} {
		assert.Equal(t, s, userStatusFromDB(userStatusToDB(s)))
	}
}

// TestCandidateIntakeEncodeDecode covers the JSONB preferences round-trip in both
// passthrough and encrypted modes, plus the null/empty degenerate forms that also
// decode into a Go string but carry no payload (CAL-117).
func TestCandidateIntakeEncodeDecode(t *testing.T) {
	intake := talent.CandidateIntake{
		TargetTitles: []string{"Staff Engineer"},
		Location:     "Accra",
		SalaryFloor:  90000,
		DealBreakers: []string{"no-relocation"},
	}

	passthrough := NewCandidateRepo(nil) // nil DBTX is fine: decode/encode never touch it
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	cipher, err := fieldcrypto.NewFieldCipher(key)
	require.NoError(t, err)
	encrypted := NewCandidateRepo(nil, WithCandidateCipher(cipher))

	for _, r := range []*CandidateRepo{passthrough, encrypted} {
		enc, err := r.encodeIntake(intake)
		require.NoError(t, err)
		got, err := r.decodeIntake(enc)
		require.NoError(t, err)
		assert.Equal(t, intake, got)
	}

	// A legacy object row (raw struct JSON) still decodes via either repo.
	legacy, err := json.Marshal(intake)
	require.NoError(t, err)
	got, err := encrypted.decodeIntake(legacy)
	require.NoError(t, err)
	assert.Equal(t, intake, got)

	// null / empty-string / empty JSONB carry no payload -> empty intake, no error
	// (they must not be mis-read as an undecryptable wrapped value).
	for _, prefs := range [][]byte{[]byte("null"), []byte(`""`), nil, {}} {
		got, err := encrypted.decodeIntake(prefs)
		require.NoError(t, err, "prefs %q must decode without error", string(prefs))
		assert.Equal(t, talent.CandidateIntake{}, got)
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
