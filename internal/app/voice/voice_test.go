package voice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/app/voice"
)

type fakeSTT struct {
	text string
	err  error
}

func (f fakeSTT) Transcribe(context.Context, []byte) (string, error) { return f.text, f.err }

type fakeTTS struct {
	audio []byte
	err   error
}

func (f fakeTTS) Synthesize(context.Context, string) ([]byte, error) { return f.audio, f.err }

func TestSpeakQuestion_Success(t *testing.T) {
	g := voice.NewGateway(nil, fakeTTS{audio: []byte("audio")}, nil)
	audio, degraded := g.SpeakQuestion(context.Background(), "Tell me about a project.")
	assert.False(t, degraded)
	assert.Equal(t, []byte("audio"), audio)
	assert.True(t, g.Enabled())
}

func TestSpeakQuestion_DegradesOnFailureOrAbsence(t *testing.T) {
	// TTS provider errors -> degrade to text, no error surfaced.
	g := voice.NewGateway(nil, fakeTTS{err: errors.New("provider down")}, nil)
	_, degraded := g.SpeakQuestion(context.Background(), "q")
	assert.True(t, degraded, "a TTS failure degrades to text")

	// No TTS wired -> degrade.
	none := voice.NewGateway(nil, nil, nil)
	_, degraded = none.SpeakQuestion(context.Background(), "q")
	assert.True(t, degraded)
	assert.False(t, none.Enabled())

	// Empty question -> degrade (nothing to speak).
	g2 := voice.NewGateway(nil, fakeTTS{audio: []byte("x")}, nil)
	_, degraded = g2.SpeakQuestion(context.Background(), "")
	assert.True(t, degraded)
}

func TestTranscribeAnswer_Success(t *testing.T) {
	g := voice.NewGateway(fakeSTT{text: "I shipped a payments API."}, nil, nil)
	text, ok := g.TranscribeAnswer(context.Background(), []byte("audio-bytes"))
	assert.True(t, ok)
	assert.Equal(t, "I shipped a payments API.", text)
}

func TestTranscribeAnswer_DegradesOnFailureOrAbsence(t *testing.T) {
	// STT provider errors -> ask the candidate to type, no error surfaced.
	g := voice.NewGateway(fakeSTT{err: errors.New("provider down")}, nil, nil)
	_, ok := g.TranscribeAnswer(context.Background(), []byte("audio"))
	assert.False(t, ok, "an STT failure degrades to typed input")

	// No STT wired / empty audio -> not ok.
	none := voice.NewGateway(nil, nil, nil)
	_, ok = none.TranscribeAnswer(context.Background(), []byte("audio"))
	assert.False(t, ok)
	g2 := voice.NewGateway(fakeSTT{text: "x"}, nil, nil)
	_, ok = g2.TranscribeAnswer(context.Background(), nil)
	assert.False(t, ok)
}
