package readiness

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateReportsDependencyFailure(t *testing.T) {
	checker := New(
		NamedCheck{Name: "ok", Check: Func(func(context.Context) error { return nil })},
		NamedCheck{Name: "db", Check: Func(func(context.Context) error { return errors.New("down") })},
	)

	err := checker.Check(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db: down")
}

func TestAggregateWithNoChecksIsReady(t *testing.T) {
	require.NoError(t, New().Check(t.Context()))
}

func TestRedisCheckPingsServer(t *testing.T) {
	addr, stop := startRedisStub(t, "")
	defer stop()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	require.NoError(t, Redis("redis://"+addr+"/0").Check.Check(ctx))
}

func TestRedisCheckAuthenticatesWhenPasswordPresent(t *testing.T) {
	addr, stop := startRedisStub(t, "secret")
	defer stop()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	require.NoError(t, Redis("redis://:secret@"+addr+"/0").Check.Check(ctx))
}

func TestRedisCheckRejectsBadURL(t *testing.T) {
	err := Redis("http://localhost:6379").Check.Check(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported redis scheme")
}

func startRedisStub(t *testing.T, password string) (string, func()) {
	t.Helper()
	var lc net.ListenConfig
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		done <- serveRedisStub(lis, password)
	}()
	stop := func() {
		_ = lis.Close()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, net.ErrClosed) {
				require.NoError(t, err)
			}
		case <-time.After(time.Second):
		}
	}
	return lis.Addr().String(), stop
}

func serveRedisStub(lis net.Listener, password string) error {
	conn, err := lis.Accept()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	reader := bufio.NewReader(conn)
	if password != "" {
		args, readErr := readRESPCommand(reader)
		if readErr != nil {
			return readErr
		}
		if len(args) != 2 || strings.ToUpper(args[0]) != "AUTH" || args[1] != password {
			_, _ = conn.Write([]byte("-ERR invalid auth\r\n"))
			return nil
		}
		_, _ = conn.Write([]byte("+OK\r\n"))
	}
	args, readErr := readRESPCommand(reader)
	if readErr != nil {
		return readErr
	}
	if len(args) == 1 && strings.ToUpper(args[0]) == "PING" {
		_, _ = conn.Write([]byte("+PONG\r\n"))
	}
	return nil
}

func readRESPCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected array line, got %q", strings.TrimSpace(line))
	}
	count := 0
	if _, err = fmt.Sscanf(strings.TrimSpace(line), "*%d", &count); err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for range count {
		sizeLine, sizeErr := r.ReadString('\n')
		if sizeErr != nil {
			return nil, sizeErr
		}
		if !strings.HasPrefix(sizeLine, "$") {
			return nil, fmt.Errorf("expected bulk size, got %q", strings.TrimSpace(sizeLine))
		}
		size := 0
		if _, sizeErr = fmt.Sscanf(strings.TrimSpace(sizeLine), "$%d", &size); sizeErr != nil {
			return nil, sizeErr
		}
		buf := make([]byte, size+2)
		if _, sizeErr = io.ReadFull(r, buf); sizeErr != nil {
			return nil, sizeErr
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}
