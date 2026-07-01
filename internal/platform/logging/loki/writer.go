// Package loki implements a tiny Loki push client for structured application logs.
// It batches JSON-encoded log lines and ships them to Loki's /loki/api/v1/push
// endpoint. It is intentionally dependency-free: it only needs net/http and the
// standard Loki push payload format.
package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config controls where and how log lines are pushed to Loki.
type Config struct {
	URL           string
	BatchSize     int
	FlushInterval time.Duration
	Timeout       time.Duration
	TenantID      string
	ServiceName   string
	Env           string
}

// entry is a single log line ready to be pushed.
type entry struct {
	ns   int64
	line string
}

// Writer is an io.Writer that batches log lines and pushes them to Loki.
// It is safe for concurrent use.
type Writer struct {
	cfg     Config
	client  *http.Client
	pushURL string

	mu      sync.Mutex
	entries []entry
	timer   *time.Timer
	wg      sync.WaitGroup
}

// New creates a Loki writer. It returns an error if the configured URL is not a
// valid HTTP(S) endpoint. The writer starts no goroutines until the first Write.
func New(cfg Config) (*Writer, error) {
	if cfg.URL == "" {
		return nil, errors.New("loki: URL is required")
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "caliber"
	}

	parsed, err := url.Parse(cfg.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("loki: invalid URL %q", cfg.URL)
	}
	pushURL := strings.TrimRight(cfg.URL, "/") + "/loki/api/v1/push"

	return &Writer{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		pushURL: pushURL,
	}, nil
}

// Write implements io.Writer. Each call appends one JSON log line to the batch.
// The actual HTTP push happens asynchronously when the batch is full or the flush
// interval elapses. Write never blocks on the network.
func (w *Writer) Write(p []byte) (int, error) {
	now := time.Now().UnixNano()

	w.mu.Lock()
	w.entries = append(w.entries, entry{ns: now, line: string(p)})
	if len(w.entries) >= w.cfg.BatchSize {
		entries := w.entries
		w.entries = nil
		w.stopTimerLocked()
		w.mu.Unlock()
		w.ship(context.Background(), entries)
		return len(p), nil
	}
	if w.timer == nil {
		w.timer = time.AfterFunc(w.cfg.FlushInterval, w.flush)
	}
	w.mu.Unlock()
	return len(p), nil
}

// Close flushes any buffered entries and stops the background timer. It should be
// called on process shutdown with a context that has a timeout.
func (w *Writer) Close(ctx context.Context) error {
	w.mu.Lock()
	w.stopTimerLocked()
	entries := w.entries
	w.entries = nil
	w.mu.Unlock()

	if len(entries) == 0 {
		w.wg.Wait()
		return nil
	}
	err := w.send(ctx, entries)
	w.wg.Wait()
	return err
}

// flush is the callback run by the interval timer.
func (w *Writer) flush() {
	w.mu.Lock()
	entries := w.entries
	w.entries = nil
	w.timer = nil
	w.mu.Unlock()

	if len(entries) > 0 {
		w.ship(context.Background(), entries)
	}
}

// send pushes a batch to Loki. Errors are written to stderr rather than returned
// to the caller, because the caller is the logging path and returning an error
// would be meaningless or recursive. Close is the exception: it can observe the
// final send error.
func (w *Writer) ship(ctx context.Context, entries []entry) {
	w.wg.Go(func() {
		_ = w.send(ctx, entries)
	})
}

func (w *Writer) send(ctx context.Context, entries []entry) error {
	err := w.sendOnce(ctx, entries)
	if err != nil {
		// Best-effort surface of shipping failures without touching the logger.
		_, _ = fmt.Fprintf(os.Stderr, "loki: push failed: %v\n", err)
	}
	return err
}

func (w *Writer) sendOnce(ctx context.Context, entries []entry) error {
	payload := pushPayload{
		Streams: []stream{{
			Stream: map[string]string{
				"service": w.cfg.ServiceName,
				"env":     w.cfg.Env,
			},
			Values: make([][2]string, len(entries)),
		}},
	}
	for i, e := range entries {
		payload.Streams[0].Values[i] = [2]string{strconv.FormatInt(e.ns, 10), e.line}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.pushURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if w.cfg.TenantID != "" {
		req.Header.Set("X-Scope-Orgid", w.cfg.TenantID)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		// Success. Drain the body to reuse the connection.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(msg))
}

func (w *Writer) stopTimerLocked() {
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
}

type pushPayload struct {
	Streams []stream `json:"streams"`
}

type stream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}
