// Package embeddings holds app.Embedder adapters: an OpenAI HTTP client and a
// deterministic dev implementation that keeps the platform runnable offline.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xcreativs/caliber/internal/app"
)

const (
	defaultModel   = "text-embedding-3-small"
	defaultBaseURL = "https://api.openai.com"
	requestTimeout = 30 * time.Second
	maxErrBody     = 1 << 14
)

// OpenAI is an app.Embedder backed by the OpenAI embeddings API.
type OpenAI struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	model      string
}

type openAIConfig struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// OpenAIOption configures the OpenAI embedder.
type OpenAIOption func(*openAIConfig)

// WithOpenAIKey sets the API key.
func WithOpenAIKey(k string) OpenAIOption { return func(c *openAIConfig) { c.apiKey = k } }

// WithOpenAIBaseURL overrides the base URL (used in tests).
func WithOpenAIBaseURL(u string) OpenAIOption { return func(c *openAIConfig) { c.baseURL = u } }

// WithOpenAIModel overrides the embedding model (ignored when empty).
func WithOpenAIModel(m string) OpenAIOption {
	return func(c *openAIConfig) {
		if m != "" {
			c.model = m
		}
	}
}

// NewOpenAI builds an OpenAI embedder. Default model is text-embedding-3-small.
func NewOpenAI(opts ...OpenAIOption) *OpenAI {
	cfg := openAIConfig{baseURL: defaultBaseURL, model: defaultModel}
	for _, o := range opts {
		o(&cfg)
	}
	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: requestTimeout}
	}
	return &OpenAI{httpClient: hc, apiKey: cfg.apiKey, baseURL: cfg.baseURL, model: cfg.model}
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// Embed returns the embedding vector for text.
func (o *OpenAI) Embed(ctx context.Context, text string) ([]float32, error) {
	payload, err := json.Marshal(embedRequest{Model: o.model, Input: text})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/v1/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
		return nil, fmt.Errorf("openai embeddings: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 {
		return nil, errors.New("openai embeddings: empty response")
	}
	out := make([]float32, len(parsed.Data[0].Embedding))
	for i, v := range parsed.Data[0].Embedding {
		out[i] = float32(v)
	}
	return out, nil
}

var _ app.Embedder = (*OpenAI)(nil)
