# Voice interview mode (EPIC-22)

Voice is a **committed post-win** capability: the POC interview runs on text, and
voice is layered on without changing the interview core. This documents the
architecture that makes that possible and keeps voice from ever being the sole
path.

## Ports — voice behind an interface (CAL-160/161)

The interview state machine is text-native (questions and answers are strings).
Voice is an edge concern, isolated behind two ports (`internal/app/voice`):

```go
type SpeechToText interface { Transcribe(ctx, audio []byte) (string, error) } // CAL-160
type TextToSpeech interface { Synthesize(ctx, text string) ([]byte, error) }   // CAL-161
```

A candidate's spoken answer is transcribed to the text the interviewer scores; a
question's text is synthesized to audio the candidate hears. The POC ships
deterministic **dev adapters** (`internal/adapters/outbound/voice`) so the path
is exercisable offline; a real provider (default **OpenAI** Whisper STT + TTS)
implements the same ports post-win, behind config — the interview core is
untouched.

## Graceful degradation to text (CAL-162)

`voice.Gateway` wraps the (optional) providers so **voice failure never blocks
the interview** — the AC. Neither method returns an error:

- `SpeakQuestion(text) -> (audio, degraded)` — `degraded=true` when TTS is
  absent or fails; the UI shows the question text.
- `TranscribeAnswer(audio) -> (text, ok)` — `ok=false` when STT is absent or
  fails; the UI asks the candidate to type.

Text is always the reliable substrate, so a missing key, a provider outage, or a
bad transcription drops the session to text rather than breaking it. Unit-tested
for every failure/absence path.

## Client device handling (CAL-163)

The browser side mirrors the server contract: voice is attempted, and any device
problem degrades to typed input. The `useVoiceInput` hook surfaces explicit
states — `unsupported` (no `mediaDevices`), `denied` (permission refused),
`idle`, `recording`, `error` — and always exposes the text field as the
fallback, so mic permission prompts, no-device, and capture errors are handled
with clear states and never trap the candidate. Voice is off by default (a
feature flag), matching the text-only POC; the live audio pipeline to the STT/TTS
providers is the post-win wiring.

## Why never the sole path

Voice reliability varies with network, device, and accent. Per the product
decision (agent_plan.md §risks), voice is a stretch presentation layer over a
text interview that always works — so a demo, and a real screening, degrade
cleanly instead of failing.
