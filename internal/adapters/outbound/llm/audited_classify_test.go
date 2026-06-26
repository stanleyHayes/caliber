package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/app/prompts"
)

// TestAudited_RecordsRegistryPromptIDAndVersion drives every registered prompt
// through the Audited decorator and asserts the recorded operation equals the
// prompt id and that a version is captured — proving CAL-032's acceptance
// criterion (prompt id + version recorded on every call) end-to-end from the
// single registry source of truth, with no fragile substring classification.
func TestAudited_RecordsRegistryPromptIDAndVersion(t *testing.T) {
	for _, p := range prompts.All() {
		rec := llm.NewMemoryRecorder(1)
		a := llm.NewAudited(stubLLM{}, rec, "dev", nil)
		_, _ = a.Complete(context.Background(), p.Request(""))

		snap := rec.Snapshot()
		require.Len(t, snap, 1)
		assert.Equalf(t, string(p.ID), snap[0].Operation, "operation for %s", p.ID)
		assert.Equalf(t, string(p.ID), snap[0].PromptID, "prompt id for %s", p.ID)
		assert.Equalf(t, p.Version, snap[0].PromptVersion, "version recorded for %s", p.ID)
		assert.NotEmptyf(t, snap[0].PromptVersion, "version is non-empty for %s", p.ID)
	}
}
