package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunProducesBackupRecording(t *testing.T) {
	dir := t.TempDir()
	cfg := captureConfig{
		OutPath:      filepath.Join(dir, "backup.json"),
		MaxQuestions: 4,
		RoleTitle:    "Senior Backend Engineer",
		Candidate:    "Ama Mensah",
		Location:     "Accra, Ghana",
	}

	require.NoError(t, run(cfg))

	b, err := os.ReadFile(cfg.OutPath)
	require.NoError(t, err)

	var rec backupRecording
	require.NoError(t, json.Unmarshal(b, &rec))

	assert.Equal(t, cfg.RoleTitle, rec.Meta.RoleTitle)
	assert.Equal(t, cfg.Candidate, rec.Meta.Candidate)
	assert.Equal(t, "text", rec.Meta.Mode)
	assert.Equal(t, cfg.MaxQuestions, rec.Meta.MaxQuestions)
	assert.NotEmpty(t, rec.Meta.GeneratedAt)

	require.Len(t, rec.Transcript, cfg.MaxQuestions)
	for i, turn := range rec.Transcript {
		assert.Equal(t, i+1, turn.Ordinal)
		assert.NotEmpty(t, turn.CompetencyTag)
		assert.NotEmpty(t, turn.Question)
		assert.NotEmpty(t, turn.Answer)
	}

	assert.NotEmpty(t, rec.ReportCard.Verdict)
	assert.NotEmpty(t, rec.ReportCard.Confidence)
	assert.NotEmpty(t, rec.ReportCard.RecommendedNextStep)
	require.NotEmpty(t, rec.ReportCard.Scores)
	for _, sc := range rec.ReportCard.Scores {
		assert.NotEmpty(t, sc.Competency)
		assert.GreaterOrEqual(t, sc.Score, 0.0)
		assert.LessOrEqual(t, sc.Score, 5.0)
		assert.NotEmpty(t, sc.Evidence, "every score cites evidence")
	}
}

func TestRunProducesDeterministicOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := captureConfig{
		OutPath:      filepath.Join(dir, "backup.json"),
		MaxQuestions: 2,
		RoleTitle:    "Platform Engineer",
		Candidate:    "Kofi Asante",
		Location:     "Tema, Ghana",
	}

	require.NoError(t, run(cfg))
	first, err := os.ReadFile(cfg.OutPath)
	require.NoError(t, err)

	require.NoError(t, run(cfg))
	second, err := os.ReadFile(cfg.OutPath)
	require.NoError(t, err)

	assert.Equal(t, string(first), string(second), "dev provider produces deterministic output")
}

func TestCaptureInterviewFailsWithZeroMaxQuestions(t *testing.T) {
	dir := t.TempDir()
	cfg := captureConfig{
		OutPath:      filepath.Join(dir, "backup.json"),
		MaxQuestions: 0,
		RoleTitle:    "Senior Backend Engineer",
		Candidate:    "Ama Mensah",
		Location:     "Accra, Ghana",
	}

	err := run(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max questions")
}
