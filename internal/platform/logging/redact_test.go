package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// newCapture returns a redacting logger writing JSON into buf, as New wires it.
func newCapture(buf *bytes.Buffer) *slog.Logger {
	return slog.New(newRedactingHandler(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
}

// decode parses the single JSON log line written to buf.
func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m); err != nil {
		t.Fatalf("log line is not valid JSON: %v\n%s", err, buf.String())
	}
	return m
}

func TestRedacts_SensitiveKeys(t *testing.T) {
	var buf bytes.Buffer
	newCapture(&buf).Info("login", "email", "ama@example.com", "password", "hunter2", "user_email", "kofi@x.io")
	m := decode(t, &buf)

	for _, key := range []string{"email", "password", "user_email"} {
		if m[key] != redactedPlaceholder {
			t.Errorf("key %q = %v, want redacted", key, m[key])
		}
	}
}

func TestRedacts_PIIShapedValues(t *testing.T) {
	var buf bytes.Buffer
	// Neutral keys, but the values carry PII-shaped substrings that must be masked
	// wherever they appear.
	newCapture(&buf).Error("request failed",
		"detail", "could not find user ama@example.com in tenant",
		"header", "Bearer abc.def.ghi123",
		"jwt", "eyJhbGci.eyJzdWIi.sig_naturE")
	m := decode(t, &buf)

	if v, _ := m["detail"].(string); strings.Contains(v, "ama@example.com") || !strings.Contains(v, redactedPlaceholder) {
		t.Errorf("email not masked in detail: %q", v)
	}
	if v, _ := m["header"].(string); strings.Contains(v, "abc.def.ghi123") || !strings.Contains(v, redactedPlaceholder) {
		t.Errorf("bearer token not masked: %q", v)
	}
	if v, _ := m["jwt"].(string); strings.Contains(v, "eyJhbGci") || !strings.Contains(v, redactedPlaceholder) {
		t.Errorf("JWT not masked: %q", v)
	}
}

func TestRedacts_MessageAndNestedGroups(t *testing.T) {
	var buf bytes.Buffer
	newCapture(&buf).Info(
		"sending to ama@example.com",
		slog.Group("actor", slog.String("email", "kofi@x.io"), slog.String("role", "EMPLOYER")),
	)
	m := decode(t, &buf)

	if msg, _ := m["msg"].(string); strings.Contains(msg, "ama@example.com") {
		t.Errorf("email leaked in message: %q", msg)
	}
	actor, ok := m["actor"].(map[string]any)
	if !ok {
		t.Fatalf("actor group missing: %v", m["actor"])
	}
	if actor["email"] != redactedPlaceholder {
		t.Errorf("nested email not redacted: %v", actor["email"])
	}
	if actor["role"] != "EMPLOYER" {
		t.Errorf("non-PII nested field changed: %v", actor["role"])
	}
}

func TestRedacts_PreservesNeutralFields(t *testing.T) {
	var buf bytes.Buffer
	// Neutral keys that merely contain "token"/"name" must NOT be over-masked.
	newCapture(&buf).Info("metrics", "service_name", "api", "token_count", float64(1234), "status", 200)
	m := decode(t, &buf)

	if m["service_name"] != "api" {
		t.Errorf("service_name over-redacted: %v", m["service_name"])
	}
	if m["token_count"] != float64(1234) {
		t.Errorf("token_count over-redacted: %v", m["token_count"])
	}
	if m["status"] != float64(200) {
		t.Errorf("status changed: %v", m["status"])
	}
}

func TestRedacts_ThroughWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	// Attributes bound up front via With must be scrubbed too.
	newCapture(&buf).With("email", "ama@example.com", "request_id", "req-1").Info("handled")
	m := decode(t, &buf)

	if m["email"] != redactedPlaceholder {
		t.Errorf("bound email not redacted: %v", m["email"])
	}
	if m["request_id"] != "req-1" {
		t.Errorf("bound neutral field changed: %v", m["request_id"])
	}
}
