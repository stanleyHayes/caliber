package kernel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestTokens(t *testing.T) {
	assert.Equal(t, []string{"c++"}, kernel.Tokens("c++"), "punctuation kept in-token")
	assert.Equal(t, []string{"c++", "systems"}, kernel.Tokens("c++ / systems"))
	assert.Equal(t, []string{".net", "core"}, kernel.Tokens(".net core"))
	assert.Equal(t, []string{"go", "sql"}, kernel.Tokens("go, sql"))
	assert.Equal(t, []string{"node.js"}, kernel.Tokens("node.js"))
	assert.Empty(t, kernel.Tokens("   "))
}
