// Command api runs the Caliber backend (gRPC + REST gateway). Wiring only;
// lifecycle lives in internal/platform/server and routing in httpserver.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"
	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	queueadapter "github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	"github.com/xcreativs/caliber/internal/app"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	contestapp "github.com/xcreativs/caliber/internal/app/contest"
	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/app/provisioning"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
	"github.com/xcreativs/caliber/internal/platform/readiness"
	"github.com/xcreativs/caliber/internal/platform/server"
	"github.com/xcreativs/caliber/internal/platform/wiring"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(cfg.LogLevel)
	if missing := cfg.Validate(); len(missing) > 0 {
		log.Warn("missing configuration", "missing", missing, "env", cfg.Env)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc, cleanup, ready, err := buildServices(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	return server.Run(ctx, cfg, log, svc, ready)
}

func buildServices(ctx context.Context, cfg config.Config, log *slog.Logger) (grpcadapter.Services, func(), *readiness.Aggregate, error) {
	model := wiring.BuildLLM(cfg, log)
	embedder := wiring.BuildEmbedder(cfg, log)
	cleanup := func() {}
	svc := grpcadapter.Services{}
	ready := readiness.New()
	if cfg.IsProd() && cfg.DatabaseURL == "" {
		return svc, cleanup, ready, errors.New("CALIBER_DATABASE_URL is required in production")
	}

	auditRepo := memory.NewAuditRepo()
	repos, repoCleanup, checks, err := wiring.OpenRepositories(ctx, cfg, log)
	if err != nil {
		return svc, cleanup, ready, err
	}
	cleanup = repoCleanup
	dispatcher, checks, closeDispatcher, err := openTaskDispatcher(cfg, checks)
	if err != nil {
		return svc, cleanup, ready, err
	}
	cleanup = func() {
		repoCleanup()
		closeDispatcher()
	}
	ready = readiness.New(checks...)

	shortlister := matchingapp.NewShortlister(
		repos.Roles, repos.Candidates, repos.Profiles,
		recallerFor(cfg, repos), embedder, model, repos.Matches)
	rejections := matchingapp.NewRejectionRecorder(repos.Roles, auditRepo, time.Now)
	matchServer := grpcadapter.NewMatchServer(shortlister, matchingapp.NewRefiner(repos.Roles, shortlister), rejections)
	svc.Match = matchServer

	tokens, terr := buildTokenService(cfg, log)
	if terr != nil {
		return svc, cleanup, ready, terr
	}
	wireApplicationServices(&svc, cfg, repos, model, auditRepo, dispatcher, matchServer, tokens)
	return svc, cleanup, ready, nil
}

func wireApplicationServices(
	svc *grpcadapter.Services,
	cfg config.Config,
	repos wiring.Repositories,
	model app.LLMClient,
	auditRepo audit.AuditRepository,
	dispatcher appqueue.TaskDispatcher,
	matchServer *grpcadapter.MatchServer,
	tokens app.TokenService,
) {
	idOpts := []identityapp.Option{
		identityapp.WithProvisioner(provisioning.NewCandidateProvisioner(repos.Candidates)),
		identityapp.WithThrottle(memory.NewLoginThrottle(time.Now, 0, 0, 0)),
	}
	identitySvc := identityapp.NewService(repos.Users, authadapter.NewArgon2idHasher(), tokens, repos.Refresh, time.Now, idOpts...)
	svc.Identity = grpcadapter.NewIdentityServer(identitySvc)
	svc.AccessVerifier = tokens
	svc.RateLimiter = grpcadapter.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, time.Now)

	svc.Role = grpcadapter.NewRoleServer(
		roles.NewSpecGenerator(model, repos.Roles, time.Now),
		roles.NewSpecEditor(repos.Roles),
		matchServer.AvailabilityCounter(),
	)
	svc.Interview = grpcadapter.NewInterviewServer(
		interviewapp.NewInterviewer(repos.Roles, repos.Interviews, model, 0, interviewapp.WithPassportUpdater(repos.Profiles)),
	)
	svc.Talent = grpcadapter.NewTalentServer(profilesapp.NewProfileBuilder(repos.Candidates, repos.Profiles, model))
	svc.Agent = grpcadapter.NewAgentServer(
		candidateagentapp.NewAgentRunner(repos.Candidates, repos.Profiles, repos.Roles, repos.Apps, model,
			candidateagentapp.WithAuditTrail(auditRepo, time.Now),
			candidateagentapp.WithWakeUpInsights(repos.Interviews, repos.Matches)),
		repos.Apps,
		dispatcher,
	)
	svc.Dashboard = grpcadapter.NewDashboardServer(dashboardapp.NewAggregator(repos.Candidates, repos.Profiles, repos.Users, repos.Roles))
	svc.Contest = grpcadapter.NewContestServer(contestapp.NewService(memory.NewContestRepo(), auditRepo, time.Now))
	svc.Audit = grpcadapter.NewAuditServer(auditRepo)
}

//nolint:ireturn // selects the concrete task-dispatch adapter for the queue port.
func openTaskDispatcher(
	cfg config.Config,
	checks []readiness.NamedCheck,
) (appqueue.TaskDispatcher, []readiness.NamedCheck, func(), error) {
	var dispatcher appqueue.TaskDispatcher = queueadapter.NewNoop()
	closeDispatcher := func() {}
	if cfg.RedisURL == "" {
		return dispatcher, checks, closeDispatcher, nil
	}
	checks = append(checks, readiness.Redis(cfg.RedisURL))
	dispatcher, err := queueadapter.NewDispatcher(cfg.RedisURL)
	if err != nil {
		return nil, checks, closeDispatcher, err
	}
	return dispatcher, checks, func() { _ = dispatcher.Close() }, nil
}

//nolint:ireturn // selects the concrete recaller adapter for the matching port.
func recallerFor(cfg config.Config, repos wiring.Repositories) matchingapp.CandidateRecaller {
	if cfg.DatabaseURL == "" || repos.Pool == nil {
		return memory.NewRecaller(repos.Candidates)
	}
	return postgres.NewRecaller(repos.Pool)
}

//nolint:ireturn // selects/constructs the concrete TokenService; interface return is intentional.
func buildTokenService(cfg config.Config, log *slog.Logger) (app.TokenService, error) {
	secret := cfg.JWTSecret
	if len(secret) < 32 {
		if cfg.IsProd() {
			return nil, errors.New("CALIBER_JWT_SECRET must be at least 32 bytes in production")
		}
		ephemeral := make([]byte, 32)
		if _, err := rand.Read(ephemeral); err != nil {
			return nil, err
		}
		secret = hex.EncodeToString(ephemeral)
		log.Warn("CALIBER_JWT_SECRET unset/weak; using an ephemeral dev secret (tokens reset on restart)")
	}
	return authadapter.NewJWTService(authadapter.JWTConfig{
		Secret: secret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience,
		AccessTTL: cfg.AccessTokenTTL, RefreshTTL: cfg.RefreshTokenTTL,
	})
}
