package kernel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestHasPhrase(t *testing.T) {
	haystack := []string{"participate", "in", "on-call", "rotation"}
	assert.True(t, kernel.HasPhrase(haystack, []string{"on-call"}))
	assert.True(t, kernel.HasPhrase([]string{"some", "night", "shift", "work"}, []string{"night", "shift"}))
	assert.False(t, kernel.HasPhrase(haystack, []string{"call"}), "whole-token match only")
	assert.False(t, kernel.HasPhrase(haystack, nil), "empty phrase never matches")
	assert.False(t, kernel.HasPhrase([]string{"go"}, []string{"go", "sql"}), "phrase longer than haystack")
	assert.True(t, kernel.HasPhrase([]string{"a", "b", "c"}, []string{"b", "c"}))
}
