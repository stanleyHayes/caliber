package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xcreativs/caliber/internal/app"
)

func TestDevCompleteReturnsParseableSpec(t *testing.T) {
	resp, err := NewDev().Complete(context.Background(), app.LLMRequest{Prompt: "Need a data engineer\nmore detail"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(resp.Text), &doc); err != nil {
		t.Fatalf("dev output not JSON: %v", err)
	}
	if doc["title"] != "Need a data engineer" {
		t.Errorf("title = %v, want first line of prompt", doc["title"])
	}
	if _, ok := doc["rubric"]; !ok {
		t.Error("missing rubric")
	}
}

func TestDevCompleteEmptyPrompt(t *testing.T) {
	resp, _ := NewDev().Complete(context.Background(), app.LLMRequest{Prompt: "   "})
	var doc map[string]any
	_ = json.Unmarshal([]byte(resp.Text), &doc)
	if doc["title"] != "Software Engineer" {
		t.Errorf("empty prompt title = %v, want default", doc["title"])
	}
}
