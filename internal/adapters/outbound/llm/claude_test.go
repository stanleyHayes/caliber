package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xcreativs/caliber/internal/app"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeCompleteConcatenatesText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-4-8",
			"content":[{"type":"text","text":"hello "},{"type":"text","text":"world"}],
			"stop_reason":"end_turn","stop_sequence":null,
			"usage":{"input_tokens":5,"output_tokens":2}
		}`))
	}))
	defer srv.Close()

	c := NewClaude(WithAPIKey("test"), WithBaseURL(srv.URL), WithModel("claude-opus-4-8"))
	resp, err := c.Complete(context.Background(), app.LLMRequest{System: "sys", Prompt: "hi", MaxTokens: 100})
	require.NoError(t, err)
	assert.Equal(t, "hello world", resp.Text)
}

func TestClaudeCompleteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"bad"}}`))
	}))
	defer srv.Close()

	c := NewClaude(WithAPIKey("test"), WithBaseURL(srv.URL))
	_, err := c.Complete(context.Background(), app.LLMRequest{Prompt: "hi"})
	require.Error(t, err)
}
