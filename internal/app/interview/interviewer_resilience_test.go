package interview_test

// Chaos & resilience tests for the interview path (Flow B, the centrepiece).
//
// These tests inject dependency failures — LLM transport errors, a persistence
// port that errors — and assert that the interview use-case degrades gracefully:
//   - it NEVER panics on a failing dependency;
//   - it NEVER writes a partial or fabricated report card (the no-fabrication
//     invariant must hold on the failure path, not just the happy path);
//   - it surfaces a clean, typed error the inbound adapter can map.
//
// They also pin the CURRENT, real behaviour of the answer-then-generate path so
// the known data-loss gap (an in-flight answer is not persisted when the next
// LLM call fails) is captured in an executable test rather than left implicit.
// See docs/chaos-testing.md for the coverage matrix and the documented gap.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// errModelDown is the canonical mid-interview LLM transport failure.
var errModelDown = errors.New("llm: provider unavailable")

// TestStartLLMFailureDoesNotPersistInterview proves that when the LLM errors
// while generating the first question, Start surfaces the error and NEVER calls
// Create — so no half-open interview row is written (no partial write). The mock
// controller fails the test if Create is invoked, because no EXPECT is set for it.
func TestStartLLMFailureDoesNotPersistInterview(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	gomock.InOrder(
		d.llm.EXPECT().Warm(gomock.Any()).Return(nil),
		d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{}, errModelDown),
	)
	// No d.interviews.Create EXPECT: the strict mock would fail if it were called.

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.DefaultConfig())
	iv, q, err := interviewer.Start(context.Background(), rl.ID, kernel.NewID(), interviewdom.ModeText)
	require.Error(t, err, "an LLM failure must surface as an error, not a silent success")
	assert.Nil(t, iv, "no interview is returned on the failure path")
	assert.Nil(t, q, "no question is returned on the failure path")
}

// TestStartPersistFailureSurfacesCleanly proves that when the LLM succeeds but
// the interview repository is unavailable (e.g. DB pool closed), Start returns a
// clean error and no successful interview/question pair is exposed to the caller.
func TestStartPersistFailureSurfacesCleanly(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	gomock.InOrder(
		d.llm.EXPECT().Warm(gomock.Any()).Return(nil),
		d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: questionJSON}, nil),
	)
	d.interviews.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("postgres: pool closed"))

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.DefaultConfig())
	iv, q, err := interviewer.Start(context.Background(), rl.ID, kernel.NewID(), interviewdom.ModeText)
	require.Error(t, err)
	assert.Nil(t, iv, "no interview is returned when persistence fails")
	assert.Nil(t, q, "no question is exposed as durable when persistence fails")
}

// TestScoreAsyncLLMFailureLeavesNoReport proves that when the scoring LLM call
// fails, ScoreAsync returns the error and the interview is left WITHOUT a report
// card (no fabricated verdict/scores are written). The strict mock guarantees
// Update is never called, so nothing is persisted in a half-scored state.
func TestScoreAsyncLLMFailureLeavesNoReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	require.NoError(t, iv.Answer("I built a payments service in Go."))

	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{}, errModelDown)
	// No Update EXPECT: a failed scoring must not persist anything.

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.DefaultConfig())
	report, err := interviewer.ScoreAsync(context.Background(), iv.ID)
	require.Error(t, err, "a scoring LLM failure must surface")
	assert.Nil(t, report, "no report card is returned on the failure path")
	assert.Nil(t, iv.Report, "no fabricated report card is attached to the interview")
	assert.NotEqual(t, interviewdom.StateClosed, iv.State, "a failed scoring must not close the interview")
}

// TestScoreAsyncTransportFailureIsTypedInternal proves the failure error is a
// clean, typed kernel error (KindInternal from DecodeJSON's transport wrap), so
// the inbound gRPC adapter maps it deterministically (to codes.Internal) rather
// than panicking or leaking raw provider detail as an untyped error.
func TestScoreAsyncTransportFailureIsTypedInternal(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	require.NoError(t, iv.Answer("A concrete answer."))

	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{}, errModelDown)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.DefaultConfig())
	_, err := interviewer.ScoreAsync(context.Background(), iv.ID)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInternal, kernel.KindOf(err),
		"a transport failure classifies as KindInternal so adapters map it deterministically")
}

// TestAnswerLLMFailureOnNextQuestionDropsInFlightAnswer pins the CURRENT (real,
// not aspirational) behaviour and the documented data-loss GAP: when the answer
// is recorded in-memory and the NEXT question's LLM call then fails, Answer
// returns the error but never reaches interviews.Update — so the just-recorded
// answer is NOT persisted. The candidate must re-submit it.
//
// This is asserted honestly against the code as it stands (interviewer.go:
// Answer records the turn, then ask()/finish() run BEFORE Update). If a durable
// fallback is added later (persist-before-generate), this test's Update
// expectation should flip to required. See docs/chaos-testing.md GAP-1.
func TestAnswerLLMFailureOnNextQuestionDropsInFlightAnswer(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)

	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	// The next-question LLM call fails. With DefaultConfig (MaxQuestions=4) and
	// one recorded turn, the use-case tries to ask another question, not finish.
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{}, errModelDown)
	// Deliberately NO Update EXPECT: the strict mock proves Update is NOT called,
	// which is exactly the data-loss gap this test documents.

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.DefaultConfig())
	pending, report, err := interviewer.Answer(context.Background(), iv.ID, "I shipped a Go payments service.")
	require.Error(t, err, "the LLM failure must surface cleanly, never panic")
	assert.Nil(t, pending)
	assert.Nil(t, report, "no fabricated question or report is produced on the failure path")

	// The turn is recorded in the in-memory aggregate...
	require.Len(t, iv.Turns, 1)
	assert.Equal(t, "I shipped a Go payments service.", iv.Turns[0].Answer)
	// ...but because Update was never reached, that answer is NOT durable. This
	// assertion documents the gap: the aggregate still holds it, yet nothing was
	// flushed to the repository.
	assert.Nil(t, iv.Pending, "the pending question was consumed when the answer was recorded")
}

// TestAnswerLLMFailureNeverFabricatesReport proves the no-fabrication invariant
// on the scoring failure path: when the final-turn scoring LLM call fails, the
// interview does NOT receive a fabricated advance/decline verdict — Report stays
// nil and the state does not advance to Closed.
func TestAnswerLLMFailureNeverFabricatesReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)

	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	// MaxQuestions=1 -> the first answer triggers scoring (finish), whose LLM
	// call we fail.
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{}, errModelDown)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.Config{MaxQuestions: 1})
	pending, report, err := interviewer.Answer(context.Background(), iv.ID, "um, not sure, various stuff")
	require.Error(t, err)
	assert.Nil(t, pending)
	assert.Nil(t, report, "a failed scoring must never yield a fabricated report card")
	assert.Nil(t, iv.Report)
	assert.NotEqual(t, interviewdom.StateClosed, iv.State, "the interview must not be closed on a scoring failure")
}

// TestAnswerMalformedScoringNeverFabricatesReport proves that even when the LLM
// "succeeds" at the transport layer but returns unparseable content on every
// attempt, the use-case returns a typed KindInvalid error and NO report card is
// fabricated from the garbage — the no-fabrication guardrail holds against
// malformed output, not just transport failure.
func TestAnswerMalformedScoringNeverFabricatesReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)

	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).
		Return(app.LLMResponse{Text: "<<garbage not json>>"}, nil).Times(app.DefaultLLMAttempts)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.Config{MaxQuestions: 1})
	_, report, err := interviewer.Answer(context.Background(), iv.ID, "a concrete answer")
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err),
		"exhausting re-asks on malformed output classifies as KindInvalid")
	assert.Nil(t, report, "malformed model output must never be shaped into a report card")
	assert.Nil(t, iv.Report)
}

// TestStartBudgetExhaustionFailsFastCleanly proves that a budget/rate-limit
// signal surfaced by the Guarded client as KindTooManyRequests during the
// pre-warm propagates as a clean, typed error (mapped to gRPC ResourceExhausted)
// and never starts nor persists an interview.
func TestStartBudgetExhaustionFailsFastCleanly(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Warm(gomock.Any()).Return(kernel.TooManyRequests("llm: cost budget exhausted; retry later"))
	// No Complete, no Create: budget exhaustion must fail fast before either.

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, interviewdom.DefaultConfig())
	_, _, err := interviewer.Start(context.Background(), rl.ID, kernel.NewID(), interviewdom.ModeText)
	require.Error(t, err)
	assert.Equal(t, kernel.KindTooManyRequests, kernel.KindOf(err),
		"budget exhaustion must surface as KindTooManyRequests, not a generic failure")
}
