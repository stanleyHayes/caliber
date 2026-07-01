package server

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunServesHealthThenShutsDown(t *testing.T) {
	cfg := config.Config{
		Env: "dev", LogLevel: "error",
		HTTPAddr: "127.0.0.1:18080", GRPCAddr: "127.0.0.1:19090",
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, logging.New("error"), grpcadapter.Services{}) }()

	var resp *http.Response
	var err error
	for range 100 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:18080/healthz", nil)
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
	cfg := config.Config{
		Env: "dev", LogLevel: "error",
		HTTPAddr: "127.0.0.1:18081", GRPCAddr: "127.0.0.1:19091",
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, cfg, logging.New("error"), grpcadapter.Services{}, readyFunc(func(context.Context) error { return nil }))
	}()

	var resp *http.Response
	var err error
	for range 100 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:18081/readyz", nil)
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
	cfg := config.Config{
		Env: "dev", LogLevel: "error",
		HTTPAddr: "127.0.0.1:18082", GRPCAddr: "127.0.0.1:19092",
	}
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
	go func() {
		opts := []Option{
			WithMetrics(metrics),
			WithAsynqmon("/asynqmon", asynq, &fakeTokenService{role: identity.RoleEmployer}),
		}
		done <- RunWithOptions(ctx, cfg, logging.New("error"), grpcadapter.Services{}, nil, opts)
	}()

	waitForOK := func(path string) {
		var resp *http.Response
		var err error
		for range 100 {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:18082"+path, nil)
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

func TestBuildRouterWithMetricsOnly(t *testing.T) {
	metrics := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r := buildRouter(runtime.NewServeMux(), config.Config{Env: "dev"}, nil, nil, runConfig{metrics: metrics})
	require.NotNil(t, r)
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
