package server

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"
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
