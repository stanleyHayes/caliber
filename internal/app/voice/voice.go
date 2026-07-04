// Package voice is the optional voice layer for the screening interview
// (EPIC-22). Speech-to-text and text-to-speech sit behind ports (CAL-160/161)
// so the interview core stays text-native, and the Gateway degrades to text on
// ANY voice failure so voice never blocks the interview (CAL-162). The POC ships
// deterministic dev adapters; a real provider (default OpenAI STT/TTS) plugs in
// behind these ports post-win.
package voice

import (
	"context"
	"log/slog"
)

// SpeechToText transcribes spoken audio to text (CAL-160): a candidate's spoken
// answer becomes the text the interviewer scores.
type SpeechToText interface {
	Transcribe(ctx context.Context, audio []byte) (string, error)
}

// TextToSpeech synthesizes speech audio from text (CAL-161): a question becomes
// audio the candidate hears.
type TextToSpeech interface {
	Synthesize(ctx context.Context, text string) ([]byte, error)
}

// Gateway wraps the optional STT/TTS providers with fail-safe degradation to
// text (CAL-162). Neither method ever returns an error: when voice is
// unavailable or fails, the caller falls back to the text path (show the
// question text / ask the candidate to type), so a voice failure can never block
// or fail the interview — text is always the reliable substrate.
type Gateway struct {
	stt SpeechToText
	tts TextToSpeech
	log *slog.Logger
}

// NewGateway wires the gateway. A nil stt/tts means that channel is unavailable
// (the caller degrades to text); a nil logger disables the degradation warning.
func NewGateway(stt SpeechToText, tts TextToSpeech, log *slog.Logger) *Gateway {
	return &Gateway{stt: stt, tts: tts, log: log}
}

// Enabled reports whether any voice channel is configured.
func (g *Gateway) Enabled() bool { return g.stt != nil || g.tts != nil }

// SpeakQuestion returns synthesized audio for a question. The bool is
// "degraded": true when TTS is unavailable or failed — the UI then presents the
// question as text.
func (g *Gateway) SpeakQuestion(ctx context.Context, text string) ([]byte, bool) {
	if g.tts == nil || text == "" {
		return nil, true
	}
	audio, err := g.tts.Synthesize(ctx, text)
	if err != nil || len(audio) == 0 {
		g.logDegrade("tts", err)
		return nil, true
	}
	return audio, false
}

// TranscribeAnswer returns the transcript of a spoken answer. The bool is "ok":
// false when STT is unavailable or failed — the UI then asks the candidate to
// type instead.
func (g *Gateway) TranscribeAnswer(ctx context.Context, audio []byte) (string, bool) {
	if g.stt == nil || len(audio) == 0 {
		return "", false
	}
	text, err := g.stt.Transcribe(ctx, audio)
	if err != nil || text == "" {
		g.logDegrade("stt", err)
		return "", false
	}
	return text, true
}

func (g *Gateway) logDegrade(channel string, err error) {
	if g.log != nil {
		// Channel + error only — never the transcript/audio — so logs stay PII-safe.
		g.log.Warn("voice degraded to text", "channel", channel, "err", err)
	}
}
