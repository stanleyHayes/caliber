package llm

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestClaudeStreamYieldsTextDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["stream"] != true {
			http.Error(w, "missing stream=true", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, "message_start", `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-4-8","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":5,"output_tokens":0}}}`)
		writeSSE(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		writeSSE(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello "}}`)
		writeSSE(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"world"}}`)
		writeSSE(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
		writeSSE(w, "message_stop", `{"type":"message_stop"}`)
	}))
	defer srv.Close()

	c := NewClaude(WithAPIKey("test"), WithBaseURL(srv.URL), WithModel("claude-opus-4-8"))
	var got string
	err := c.Stream(context.Background(), app.LLMRequest{Prompt: "hi", MaxTokens: 100}, func(ev app.LLMStreamEvent) error {
		got += ev.Text
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, "hello world", got)
}

func writeSSE(w http.ResponseWriter, event, data string) {
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}
