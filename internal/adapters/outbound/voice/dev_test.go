package voice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	voice "github.com/xcreativs/caliber/internal/adapters/outbound/voice"
)

func TestDevSTT(t *testing.T) {
	text, err := voice.DevSTT{}.Transcribe(context.Background(), []byte("audio"))
	require.NoError(t, err)
	assert.NotEmpty(t, text)
	_, err = voice.DevSTT{}.Transcribe(context.Background(), nil)
	assert.Error(t, err, "empty audio is an error the gateway degrades on")
}

func TestDevTTS(t *testing.T) {
	audio, err := voice.DevTTS{}.Synthesize(context.Background(), "hello")
	require.NoError(t, err)
	assert.NotEmpty(t, audio)
	_, err = voice.DevTTS{}.Synthesize(context.Background(), "")
	assert.Error(t, err)
}
