package jobs_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
	"github.com/xcreativs/caliber/internal/app"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"
)

func TestCandidateAgentHandlerRunsAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	roles := mocks.NewMockRoleRepository(ctrl)
	apps := mocks.NewMockApplicationRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)

	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cid, "s", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1, 0))
	require.NoError(t, err)

	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: `{"fit_score":0.85,"apply":true,"tailored_summary":"Drawing on verified Go experience."}`}, nil)
	apps.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	mux := jobs.NewMux(slog.New(slog.DiscardHandler))
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{
		AgentRunner: candidateagentapp.NewAgentRunner(candidates, profiles, roles, apps, llm),
	}, slog.New(slog.DiscardHandler))

	payload, err := json.Marshal(appqueue.CandidateAgentRunPayload{CandidateID: cid.String()})
	require.NoError(t, err)

	err = mux.ProcessTask(context.Background(), asynq.NewTask(string(appqueue.TypeCandidateAgentRun), payload))
	require.NoError(t, err)
}

func TestInterviewScoringHandlerRejectsInvalidPayload(t *testing.T) {
	mux := jobs.NewMux(slog.New(slog.DiscardHandler))
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{}, slog.New(slog.DiscardHandler))

	err := mux.ProcessTask(context.Background(), asynq.NewTask(string(appqueue.TypeInterviewScoring), []byte("not json")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode interview scoring payload")
}
