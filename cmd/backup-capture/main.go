// Command backup-capture records a clean, live-style Flow B interview transcript
// and report card as a pre-recorded backup for venue network failure (CAL-106).
//
// It drives the real interview use-case directly against the deterministic dev
// LLM provider and in-memory repositories, so it works offline and produces the
// same output every time. The resulting JSON is checked into the repo and can be
// displayed by the frontend if the live streamed path fails.
//
// Usage:
//
//	go run ./cmd/backup-capture -out web/public/interview-backup.json
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// captureConfig holds the user-controllable parts of a backup capture.
type captureConfig struct {
	OutPath      string
	MaxQuestions int
	RoleTitle    string
	Candidate    string
	Location     string
}

func parseFlags() captureConfig {
	var cfg captureConfig
	flag.StringVar(&cfg.OutPath, "out", "web/public/interview-backup.json", "path to write the captured backup JSON")
	flag.IntVar(&cfg.MaxQuestions, "questions", 4, "number of interview turns to record")
	flag.StringVar(&cfg.RoleTitle, "role", "Senior Backend Engineer", "title of the demo role")
	flag.StringVar(&cfg.Candidate, "candidate", "Ama Mensah", "name of the demo candidate")
	flag.StringVar(&cfg.Location, "location", "Accra, Ghana", "location of the demo role")
	flag.Parse()
	return cfg
}

// turnSnapshot is one question/answer exchange in the backup transcript.
type turnSnapshot struct {
	Ordinal       int    `json:"ordinal"`
	CompetencyTag string `json:"competencyTag"`
	Question      string `json:"question"`
	Answer        string `json:"answer"`
}

// scoreSnapshot is a per-competency score with evidence.
type scoreSnapshot struct {
	Competency string  `json:"competency"`
	Score      float64 `json:"score"`
	Evidence   string  `json:"evidence"`
}

// reportCardSnapshot is the scored, evidence-tagged result of the interview.
type reportCardSnapshot struct {
	Verdict             string          `json:"verdict"`
	Confidence          string          `json:"confidence"`
	RecommendedNextStep string          `json:"recommendedNextStep"`
	Scores              []scoreSnapshot `json:"scores"`
}

// backupRecording is the top-level artifact written to disk.
type backupRecording struct {
	Meta       recordingMeta      `json:"meta"`
	Transcript []turnSnapshot     `json:"transcript"`
	ReportCard reportCardSnapshot `json:"reportCard"`
}

// recordingMeta describes when and how the backup was captured.
type recordingMeta struct {
	GeneratedAt  string `json:"generatedAt"`
	RoleTitle    string `json:"roleTitle"`
	Candidate    string `json:"candidate"`
	Mode         string `json:"mode"`
	MaxQuestions int    `json:"maxQuestions"`
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "backup-capture: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg captureConfig) error {
	ctx := context.Background()

	roleRepo := memory.NewRoleRepo()
	interviewRepo := memory.NewInterviewRepo()
	profileRepo := memory.NewTalentProfileRepo()

	candidateID := kernel.NewID()
	roleID, err := seedDemoData(ctx, roleRepo, profileRepo, candidateID, cfg)
	if err != nil {
		return fmt.Errorf("seed demo data: %w", err)
	}

	recording, err := captureInterview(ctx, roleRepo, interviewRepo, profileRepo, candidateID, roleID, cfg)
	if err != nil {
		return fmt.Errorf("capture interview: %w", err)
	}

	if err := writeJSON(cfg.OutPath, recording); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Captured %d-turn interview to %s\n", len(recording.Transcript), cfg.OutPath)
	_, _ = fmt.Fprintf(os.Stdout, "Verdict: %s (%s)\n", recording.ReportCard.Verdict, recording.ReportCard.Confidence)
	return nil
}

// seedDemoData creates a believable role and candidate profile for the backup.
func seedDemoData(
	ctx context.Context,
	roleRepo *memory.RoleRepo,
	profileRepo *memory.TalentProfileRepo,
	candidateID kernel.ID,
	cfg captureConfig,
) (kernel.ID, error) {
	profile, err := talent.NewTalentProfile(candidateID,
		"Experienced backend engineer with a track record of building high-throughput services in Go.",
		[]talent.ProfileCompetency{
			{Name: "Go", Level: 4, EvidenceQuote: "Built production services in Go", SourceSpan: "CV"},
			{Name: "Postgres", Level: 4, EvidenceQuote: "Designed schemas and query optimisation", SourceSpan: "CV"},
			{Name: "gRPC", Level: 3, EvidenceQuote: "Implemented inter-service gRPC APIs", SourceSpan: "CV"},
			{Name: "Kubernetes", Level: 3, EvidenceQuote: "Deployed workloads to Kubernetes clusters", SourceSpan: "CV"},
		})
	if err != nil {
		return "", fmt.Errorf("create profile: %w", err)
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		return "", fmt.Errorf("store profile: %w", err)
	}

	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title:        cfg.RoleTitle,
			Location:     cfg.Location,
			Seniority:    role.SenioritySenior,
			Availability: "within 1 month",
			MustHaves:    []string{"Go", "Postgres"},
			NiceToHaves:  []string{"Kubernetes", "gRPC"},
			SalaryBand:   kernel.SalaryBand{Currency: "GHS", Low: 18000, High: 25000},
		},
		role.Rubric{Competencies: []role.Competency{
			{Name: "Go", Weight: 0.35, MustHave: true},
			{Name: "Postgres", Weight: 0.25, MustHave: true},
			{Name: "gRPC", Weight: 0.20, MustHave: false},
			{Name: "Kubernetes", Weight: 0.20, MustHave: false},
		}}.Normalize(),
		time.Now())
	if err != nil {
		return "", fmt.Errorf("create role: %w", err)
	}
	if err := rl.Open(); err != nil {
		return "", fmt.Errorf("open role: %w", err)
	}
	if err := roleRepo.Create(ctx, rl); err != nil {
		return "", fmt.Errorf("store role: %w", err)
	}
	return rl.ID, nil
}

// demoAnswers returns concrete, first-person responses that show honest signal pressure.
func demoAnswers() []string {
	return []string{
		"I led a team of four to rebuild the payments gateway in Go, " +
			"cut p99 latency from 400ms to 80ms, and rolled it out to production over six weeks.",
		"I designed the Postgres schema and query paths for that service, " +
			"added partial indexes on the settlement table, and reduced a slow report from minutes to under ten seconds.",
		"We used gRPC with protobuf for service-to-service calls and exposed a REST gateway " +
			"for third-party integrations; I owned the proto design and error-status mapping.",
		"I containerised the service and wrote Kubernetes deployments with HPA and " +
			"liveness/readiness probes; we ran it on a managed cluster in staging and production.",
	}
}

// captureInterview drives the interview FSM and returns a recording.
func captureInterview(
	ctx context.Context,
	roleRepo *memory.RoleRepo,
	interviewRepo *memory.InterviewRepo,
	profileRepo *memory.TalentProfileRepo,
	candidateID, roleID kernel.ID,
	cfg captureConfig,
) (backupRecording, error) {
	interviewer := interviewapp.NewInterviewer(
		roleRepo, interviewRepo, llm.NewDev(),
		interviewdom.Config{MaxQuestions: cfg.MaxQuestions},
		interviewapp.WithPassportUpdater(profileRepo),
	)

	iv, pending, err := interviewer.Start(ctx, roleID, candidateID, interviewdom.ModeText)
	if err != nil {
		return backupRecording{}, fmt.Errorf("start interview: %w", err)
	}

	recording := backupRecording{
		Meta: recordingMeta{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			RoleTitle:    cfg.RoleTitle,
			Candidate:    cfg.Candidate,
			Mode:         interviewdom.ModeText.String(),
			MaxQuestions: cfg.MaxQuestions,
		},
		Transcript: make([]turnSnapshot, 0, cfg.MaxQuestions),
	}

	for len(recording.Transcript) < cfg.MaxQuestions {
		if pending == nil {
			return backupRecording{}, fmt.Errorf("interview ended before reaching %d questions", cfg.MaxQuestions)
		}

		answers := demoAnswers()
		answer := answers[len(recording.Transcript)%len(answers)]
		recording.Transcript = append(recording.Transcript, turnSnapshot{
			Ordinal:       pending.Ordinal,
			CompetencyTag: pending.CompetencyTag,
			Question:      pending.Text,
			Answer:        answer,
		})

		next, report, err := interviewer.Answer(ctx, iv.ID, answer)
		if err != nil {
			return backupRecording{}, fmt.Errorf("answer turn %d: %w", len(recording.Transcript), err)
		}
		if report != nil {
			recording.ReportCard = reportCardSnapshot{
				Verdict:             report.Verdict.String(),
				Confidence:          report.Confidence.String(),
				RecommendedNextStep: report.RecommendedNextStep,
				Scores:              mapScores(report.Scores),
			}
			return recording, nil
		}
		pending = next
	}

	return backupRecording{}, errors.New("interview reached max questions without a report card")
}

func mapScores(scores []interviewdom.CompetencyScore) []scoreSnapshot {
	out := make([]scoreSnapshot, len(scores))
	for i, s := range scores {
		out[i] = scoreSnapshot{
			Competency: s.Competency,
			Score:      s.Score,
			Evidence:   s.Evidence,
		}
	}
	return out
}

func writeJSON(path string, v any) error {
	path = filepath.Clean(path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
