// Package voice holds voice-provider adapters (EPIC-22). The POC ships only the
// deterministic dev adapters below, which exercise the voice ports without audio
// hardware or an API key; a real provider (default OpenAI Whisper STT + TTS)
// implements the same app/voice ports post-win.
package voice

import (
	"context"
	"errors"
)

// DevSTT is a deterministic speech-to-text stub: it returns a fixed transcript
// so the voice path is testable offline. Empty audio is an error (which the
// Gateway turns into a graceful text fallback).
type DevSTT struct{}

// Transcribe returns a placeholder transcript for non-empty audio.
func (DevSTT) Transcribe(_ context.Context, audio []byte) (string, error) {
	if len(audio) == 0 {
		return "", errors.New("voice: empty audio")
	}
	return "[dev transcript]", nil
}

// DevTTS is a deterministic text-to-speech stub: it returns placeholder audio
// bytes derived from the text. Empty text is an error.
type DevTTS struct{}

// Synthesize returns placeholder audio bytes for non-empty text.
func (DevTTS) Synthesize(_ context.Context, text string) ([]byte, error) {
	if text == "" {
		return nil, errors.New("voice: empty text")
	}
	return append([]byte("dev-audio:"), text...), nil
}
