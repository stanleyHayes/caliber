package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"
	"github.com/xcreativs/caliber/internal/adapters/inbound/httpserver"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunServesHealthThenShutsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	base := startServer(ctx, t, done, nil)

	var resp *http.Response
	var err error
	for range 100 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/healthz", nil)
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{"status":"ok"}`, string(body))

	cancel()
	select {
	case runErr := <-done:
		require.NoError(t, runErr)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestRunServesReadyzWithInjectedChecks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	base := startServer(ctx, t, done, []httpserver.ReadinessChecker{readyFunc(func(context.Context) error { return nil })})

	var resp *http.Response
	var err error
	for range 100 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/readyz", nil)
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{"status":"ready"}`, string(body))

	cancel()
	select {
	case runErr := <-done:
		require.NoError(t, runErr)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestRunWithMetricsAndAsynqmon(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsHit := make(chan struct{}, 1)
	metrics := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		metricsHit <- struct{}{}
		w.WriteHeader(http.StatusOK)
	})
	asynqmonHit := make(chan struct{}, 1)
	asynq := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asynqmonHit <- struct{}{}
		w.WriteHeader(http.StatusOK)
	})

	done := make(chan error, 1)
	base := startServer(ctx, t, done, nil,
		WithMetrics(metrics),
		WithAsynqmon("/asynqmon", asynq, &fakeTokenService{role: identity.RoleEmployer}),
	)

	waitForOK := func(path string) {
		var resp *http.Response
		var err error
		for range 100 {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
			req.Header.Set("Authorization", "Bearer valid-token")
			resp, err = http.DefaultClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
			time.Sleep(20 * time.Millisecond)
		}
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		_ = resp.Body.Close()
	}

	waitForOK("/metrics")
	select {
	case <-metricsHit:
	case <-time.After(2 * time.Second):
		t.Fatal("metrics handler was not invoked")
	}

	waitForOK("/asynqmon/")
	select {
	case <-asynqmonHit:
	case <-time.After(2 * time.Second):
		t.Fatal("asynqmon handler was not invoked")
	}

	cancel()
	select {
	case runErr := <-done:
		require.NoError(t, runErr)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

// TestRunReturnsErrorWhenHTTPPortTaken covers the Run wrapper and the HTTP-bind
// failure path: gRPC binds on an ephemeral port, but the HTTP listener can't take
// an already-occupied port, so Run tears the gRPC server back down and returns.
func TestRunReturnsErrorWhenHTTPPortTaken(t *testing.T) {
	var lc net.ListenConfig
	occupied, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = occupied.Close() }()

	cfg := config.Config{
		Env: "dev", LogLevel: "error",
		HTTPAddr: occupied.Addr().String(), GRPCAddr: "127.0.0.1:0",
	}
	err = Run(context.Background(), cfg, logging.New("error"), grpcadapter.Services{})
	require.Error(t, err, "Run must return when the HTTP port cannot be bound")
}

func TestBuildRouterWithMetricsOnly(t *testing.T) {
	metrics := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r := buildRouter(runtime.NewServeMux(), config.Config{Env: "dev"}, nil, nil, runConfig{metrics: metrics})
	require.NotNil(t, r)

	// Without a verifier (local dev) /metrics stays open.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestBuildRouterGatesOperatorEndpoints is the CAL-120 fix: with a verifier
// configured, /debug/ai-quality requires an authenticated operator. /metrics
// stays open (Prometheus scrapes it; it is protected at the network layer) so
// gating it would not break scraping.
func TestBuildRouterGatesOperatorEndpoints(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r := buildRouter(runtime.NewServeMux(), config.Config{Env: "dev"}, nil, nil, runConfig{
		metrics: ok, aiQualityMetrics: ok, verifier: &fakeTokenService{role: identity.RoleEmployer},
	})

	// /debug/ai-quality: no bearer -> 401; valid operator bearer -> 200.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/ai-quality", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "/debug/ai-quality must require auth when a verifier is set")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/ai-quality", nil)
	req.Header.Set("Authorization", "Bearer valid")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "/debug/ai-quality is reachable by an authenticated operator")

	// /metrics stays open for the Prometheus scraper even with a verifier set.
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusOK, rec.Code, "/metrics is not gated behind user auth (network-protected)")
}

// startServer runs the API on OS-assigned ephemeral ports (":0") to avoid
// fixed-port collisions with other processes on a shared machine, and returns
// its HTTP base URL (e.g. "http://127.0.0.1:54321"). The Run error is delivered
// on done at shutdown.
func startServer(
	ctx context.Context,
	t *testing.T,
	done chan<- error,
	readiness []httpserver.ReadinessChecker,
	opts ...Option,
) string {
	t.Helper()
	cfg := config.Config{
		Env: "dev", LogLevel: "error",
		HTTPAddr: "127.0.0.1:0", GRPCAddr: "127.0.0.1:0",
	}
	addrCh := make(chan string, 1)
	opts = append(opts, WithListenCallback(func(a Addrs) { addrCh <- a.HTTP }))
	go func() {
		done <- RunWithOptions(ctx, cfg, logging.New("error"), grpcadapter.Services{}, readiness, opts)
	}()
	select {
	case addr := <-addrCh:
		return "http://" + addr
	case <-time.After(5 * time.Second):
		t.Fatal("server did not report its listen address in time")
		return ""
	}
}

type readyFunc func(context.Context) error

func (f readyFunc) Check(ctx context.Context) error { return f(ctx) }

type fakeTokenService struct {
	role identity.Role
	err  error
}

func (f *fakeTokenService) IssueAccess(_ app.Principal) (app.AccessToken, error) {
	return app.AccessToken{}, nil
}

func (f *fakeTokenService) IssueRefresh(_ app.Principal) (app.RefreshToken, error) {
	return app.RefreshToken{}, nil
}

func (f *fakeTokenService) VerifyAccess(_ string) (app.Principal, error) {
	if f.err != nil {
		return app.Principal{}, f.err
	}
	return app.Principal{UserID: "user-1", Role: f.role.String()}, nil
}

func (f *fakeTokenService) VerifyRefresh(_ string) (app.RefreshClaims, error) {
	return app.RefreshClaims{}, nil
}
