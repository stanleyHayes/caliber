// Package readiness provides small dependency checks for the API readiness
// endpoint. It stays in platform code because it deals with sockets and
// infrastructure health, not domain behavior.
package readiness

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

const defaultRedisPort = "6379"

// Checker reports whether a dependency is currently ready.
type Checker interface {
	Check(ctx context.Context) error
}

// Func adapts a function into a Checker.
type Func func(context.Context) error

// Check runs f.
func (f Func) Check(ctx context.Context) error { return f(ctx) }

// NamedCheck labels a dependency check for diagnostics.
type NamedCheck struct {
	Name  string
	Check Checker
}

// Aggregate runs all configured checks. A nil or empty aggregate is ready.
type Aggregate struct {
	checks []NamedCheck
}

// New builds a readiness aggregate.
func New(checks ...NamedCheck) *Aggregate {
	filtered := make([]NamedCheck, 0, len(checks))
	for _, check := range checks {
		if strings.TrimSpace(check.Name) == "" || check.Check == nil {
			continue
		}
		filtered = append(filtered, check)
	}
	return &Aggregate{checks: filtered}
}

// Check returns an error if any dependency check fails.
func (a *Aggregate) Check(ctx context.Context) error {
	if a == nil {
		return nil
	}
	var errs []error
	for _, check := range a.checks {
		if err := check.Check.Check(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", check.Name, err))
		}
	}
	return errors.Join(errs...)
}

// Redis builds a readiness check that verifies a Redis URL accepts PING.
func Redis(rawURL string) NamedCheck {
	return NamedCheck{Name: "redis", Check: Func(func(ctx context.Context) error {
		return pingRedis(ctx, rawURL)
	})}
}

func pingRedis(ctx context.Context, rawURL string) error {
	target, err := redisDialTarget(rawURL)
	if err != nil {
		return err
	}
	conn, err := dialRedis(ctx, target)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if err := setConnDeadline(ctx, conn); err != nil {
		return err
	}

	r := bufio.NewReader(conn)
	if err := authenticateRedis(conn, r, target.authArgs); err != nil {
		return err
	}
	return pingRedisConn(conn, r)
}

func dialRedis(ctx context.Context, target redisTarget) (net.Conn, error) {
	var dialer net.Dialer
	if target.tls {
		return (&tls.Dialer{NetDialer: &dialer}).DialContext(ctx, "tcp", target.host)
	}
	return dialer.DialContext(ctx, "tcp", target.host)
}

func setConnDeadline(ctx context.Context, conn net.Conn) error {
	if deadline, ok := ctx.Deadline(); ok {
		return conn.SetDeadline(deadline)
	}
	return conn.SetDeadline(time.Now().Add(2 * time.Second))
}

func authenticateRedis(conn net.Conn, r *bufio.Reader, authArgs []string) error {
	if len(authArgs) == 0 {
		return nil
	}
	if err := writeRedisCommand(conn, append([]string{"AUTH"}, authArgs...)...); err != nil {
		return err
	}
	if err := expectSimpleOK(r); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	return nil
}

func pingRedisConn(conn net.Conn, r *bufio.Reader) error {
	if err := writeRedisCommand(conn, "PING"); err != nil {
		return err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	if line != "+PONG\r\n" {
		return fmt.Errorf("unexpected ping response %q", strings.TrimSpace(line))
	}
	return nil
}

type redisTarget struct {
	host     string
	authArgs []string
	tls      bool
}

func redisDialTarget(rawURL string) (redisTarget, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return redisTarget{}, err
	}
	if u.Scheme != "redis" && u.Scheme != "rediss" {
		return redisTarget{}, fmt.Errorf("unsupported redis scheme %q", u.Scheme)
	}
	if strings.TrimSpace(u.Host) == "" {
		return redisTarget{}, errors.New("missing redis host")
	}
	return redisTarget{host: redisHost(u), authArgs: redisAuthArgs(u.User), tls: u.Scheme == "rediss"}, nil
}

func redisHost(u *url.URL) string {
	host := u.Host
	if u.Port() == "" {
		host = net.JoinHostPort(u.Hostname(), defaultRedisPort)
	}
	return host
}

func redisAuthArgs(user *url.Userinfo) []string {
	if user == nil {
		return nil
	}
	password, hasPassword := user.Password()
	username := user.Username()
	switch {
	case username != "" && hasPassword:
		return []string{username, password}
	case username != "":
		return []string{username}
	case hasPassword:
		return []string{password}
	default:
		return nil
	}
}

func writeRedisCommand(conn net.Conn, args ...string) error {
	if _, err := fmt.Fprintf(conn, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return nil
}

func expectSimpleOK(r *bufio.Reader) error {
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	if line != "+OK\r\n" {
		return fmt.Errorf("unexpected response %q", strings.TrimSpace(line))
	}
	return nil
}
