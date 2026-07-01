package loki

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(url string) Config {
	return Config{
		URL:           url,
		BatchSize:     1,
		FlushInterval: 5 * time.Second,
		Timeout:       5 * time.Second,
		ServiceName:   "caliber-test",
		Env:           "test",
		TenantID:      "tenant-42",
	}
}

func TestNewRejectsEmptyURL(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
}

func TestNewRejectsInvalidURL(t *testing.T) {
	for _, raw := range []string{"ftp://loki.example.com", "://bad"} {
		_, err := New(Config{URL: raw})
		require.Error(t, err, "URL %q should be rejected", raw)
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()

	w, err := New(Config{URL: srv.URL, ServiceName: "svc", Env: "dev"})
	require.NoError(t, err)
	assert.Equal(t, 100, w.cfg.BatchSize)
	assert.Equal(t, 5*time.Second, w.cfg.FlushInterval)
	assert.Equal(t, 10*time.Second, w.cfg.Timeout)
}

func TestWriterPushesLine(t *testing.T) {
	var got pushPayload
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/loki/api/v1/push", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "tenant-42", r.Header.Get("X-Scope-Orgid"))

		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	w, err := New(testConfig(srv.URL))
	require.NoError(t, err)

	_, err = w.Write([]byte(`{"msg":"hello"}`))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got.Streams) == 1 && len(got.Streams[0].Values) == 1
	}, 2*time.Second, 10*time.Millisecond)

	mu.Lock()
	stream := got.Streams[0].Stream
	values := got.Streams[0].Values
	mu.Unlock()
	assert.Equal(t, map[string]string{"service": "caliber-test", "env": "test"}, stream)
	assert.JSONEq(t, `{"msg":"hello"}`, values[0][1])
	assert.NotEmpty(t, values[0][0]) // nanosecond timestamp

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, w.Close(ctx))
}

func TestWriterBatchesLines(t *testing.T) {
	var calls atomic.Int32
	var got pushPayload
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		var p pushPayload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshal payload: %v", err)
		}
		mu.Lock()
		got = p
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.BatchSize = 10
	w, err := New(cfg)
	require.NoError(t, err)

	for i := range 7 {
		_, err = w.Write([]byte(`{"i":` + string(rune('0'+i)) + `}`))
		require.NoError(t, err)
	}

	// Nothing should have shipped yet; the batch is not full and the interval is long.
	assert.Equal(t, int32(0), calls.Load())
	assert.Len(t, w.entries, 7)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, w.Close(ctx))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, got.Streams, 1)
	require.Len(t, got.Streams[0].Values, 7)
	for i := range 7 {
		want := `{"i":` + string(rune('0'+i)) + `}`
		assert.JSONEq(t, want, got.Streams[0].Values[i][1])
	}
}

func TestWriterFlushesOnInterval(t *testing.T) {
	var got pushPayload
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.BatchSize = 100
	cfg.FlushInterval = 50 * time.Millisecond
	w, err := New(cfg)
	require.NoError(t, err)

	_, err = w.Write([]byte(`{"msg":"interval"}`))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got.Streams) == 1 && len(got.Streams[0].Values) == 1
	}, 2*time.Second, 10*time.Millisecond)

	mu.Lock()
	values := got.Streams[0].Values
	mu.Unlock()
	assert.JSONEq(t, `{"msg":"interval"}`, values[0][1])

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, w.Close(ctx))
}

func TestWriterFlushesOnClose(t *testing.T) {
	var got pushPayload
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.BatchSize = 100
	w, err := New(cfg)
	require.NoError(t, err)

	_, err = w.Write([]byte(`{"msg":"close"}`))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, w.Close(ctx))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, got.Streams, 1)
	assert.JSONEq(t, `{"msg":"close"}`, got.Streams[0].Values[0][1])
}

func TestWriterReportsPushError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.BatchSize = 2 // keep the line buffered so Close can observe the push error
	w, err := New(cfg)
	require.NoError(t, err)

	_, err = w.Write([]byte(`{"msg":"fail"}`))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = w.Close(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestWriterNoTenantHeaderWhenUnset(t *testing.T) {
	var header string
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		header = r.Header.Get("X-Scope-Orgid")
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.TenantID = ""
	w, err := New(cfg)
	require.NoError(t, err)

	_, err = w.Write([]byte(`{"msg":"no-tenant"}`))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return header != "tenant-42"
	}, 2*time.Second, 10*time.Millisecond)

	mu.Lock()
	h := header
	mu.Unlock()
	assert.Empty(t, h)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, w.Close(ctx))
}
