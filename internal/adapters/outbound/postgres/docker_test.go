package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

// skipIfNoDocker skips the calling integration test fast when no Docker daemon is
// reachable (e.g. a local `make test` without Docker running), instead of blocking
// on container startup. The provider probe is bounded by a timeout so it can never
// hang the suite; CI provides Docker, so the integration tests still run there.
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		p, err := testcontainers.ProviderDocker.GetProvider()
		if err != nil {
			done <- err
			return
		}
		done <- p.Health(context.Background())
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Skipf("Docker not available; skipping integration test: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Skip("Docker health probe timed out; skipping integration test")
	}
}
