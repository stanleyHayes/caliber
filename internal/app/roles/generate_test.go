package roles_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/salary"
	"github.com/xcreativs/caliber/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const validSpecJSON = `{"title":"Backend Engineer","location":"Accra","seniority":"senior","availability":"now","responsibilities":["build"],"must_haves":["Go"],"nice_to_haves":[],"salary_band":{"currency":"GHS","low":1000,"high":2000},"rubric":[{"name":"Go","weight":0.6,"must_have":true},{"name":"SQL","weight":0.4,"must_have":false}]}`

func fixedClock() app.Clock { return func() time.Time { return time.Unix(1700000000, 0) } }

func TestGenerateHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)

	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: validSpecJSON}, nil)
	var saved *role.Role
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r *role.Role) error {
		saved = r
		return nil
	})

	emp := kernel.NewID()
	r, err := roles.NewSpecGenerator(llm, repo, fixedClock()).Generate(context.Background(), emp, "senior Go engineer in Accra")
	require.NoError(t, err)
	assert.Equal(t, "Backend Engineer", r.Title)
	assert.Equal(t, emp, r.EmployerID)
	assert.False(t, r.ID.IsZero())
	assert.InDelta(t, 1.0, r.Rubric.TotalWeight(), 0.001)
	require.NotNil(t, saved)
	assert.Equal(t, r.ID, saved.ID)
}

func TestGenerateUnknownSeniorityDefaultsMid(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
		Return(app.LLMResponse{Text: `{"title":"X","seniority":"wizard","rubric":[{"name":"A","weight":1}]}`}, nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	r, err := roles.NewSpecGenerator(llm, repo, fixedClock()).Generate(context.Background(), kernel.NewID(), "x")
	require.NoError(t, err)
	assert.Equal(t, "mid", r.Spec.Seniority.String())
}

func TestGenerateInputValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	// No EXPECT(): the LLM/repo must NOT be called for invalid input.
	g := roles.NewSpecGenerator(mocks.NewMockLLMClient(ctrl), mocks.NewMockRoleRepository(ctrl), fixedClock())

	_, err := g.Generate(context.Background(), kernel.ID(""), "x")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
	_, err = g.Generate(context.Background(), kernel.NewID(), "   ")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestGenerateLLMError(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{}, errors.New("boom"))
	_, err := roles.NewSpecGenerator(llm, mocks.NewMockRoleRepository(ctrl), fixedClock()).
		Generate(context.Background(), kernel.NewID(), "x")
	require.Error(t, err)
}

func TestGenerateBadJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: "not json"}, nil).Times(app.DefaultLLMAttempts)
	_, err := roles.NewSpecGenerator(llm, mocks.NewMockRoleRepository(ctrl), fixedClock()).
		Generate(context.Background(), kernel.NewID(), "x")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestGenerateDomainValidationFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
		Return(app.LLMResponse{Text: `{"title":"X","seniority":"mid","rubric":[]}`}, nil)
	_, err := roles.NewSpecGenerator(llm, mocks.NewMockRoleRepository(ctrl), fixedClock()).
		Generate(context.Background(), kernel.NewID(), "x")
	require.Error(t, err)
}

func TestGeneratePreservesExplicitSalaryBand(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: validSpecJSON}, nil)
	var saved *role.Role
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r *role.Role) error {
		saved = r
		return nil
	})

	_, err := roles.NewSpecGenerator(llm, repo, fixedClock()).Generate(context.Background(), kernel.NewID(), "x")
	require.NoError(t, err)
	require.NotNil(t, saved)
	// A band the model supplied is kept verbatim; the market lookup never overrides it.
	assert.InDelta(t, 1000.0, saved.Spec.SalaryBand.Low, 0.001)
	assert.InDelta(t, 2000.0, saved.Spec.SalaryBand.High, 0.001)
}

func TestGenerateFillsMissingSalaryFromMarket(t *testing.T) {
	ctrl := gomock.NewController(t)
	llm := mocks.NewMockLLMClient(ctrl)
	repo := mocks.NewMockRoleRepository(ctrl)
	// A valid spec with NO salary_band -> the realism fallback fills a Ghana band.
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{
		Text: `{"title":"Backend Engineer","location":"Accra","seniority":"senior","rubric":[{"name":"Go","weight":1,"must_have":true}]}`,
	}, nil)
	var saved *role.Role
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r *role.Role) error {
		saved = r
		return nil
	})

	_, err := roles.NewSpecGenerator(llm, repo, fixedClock()).Generate(context.Background(), kernel.NewID(), "senior Go engineer")
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.False(t, saved.Spec.SalaryBand.IsZero(), "a blank band is filled, not left empty")
	assert.Equal(t, salary.Lookup("Backend Engineer", role.SenioritySenior), saved.Spec.SalaryBand)
}
