package dashboards

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDashboardsAreValid(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dashboards dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			b, err := os.ReadFile(filepath.Clean(e.Name()))
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			var dash map[string]any
			if err := json.Unmarshal(b, &dash); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if dash["title"] == "" {
				t.Error("dashboard missing title")
			}
			if dash["uid"] == "" {
				t.Error("dashboard missing uid")
			}
			panels, ok := dash["panels"].([]any)
			if !ok || len(panels) == 0 {
				t.Error("dashboard has no panels")
			}
		})
	}
}
