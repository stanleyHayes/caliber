package app

import "time"

// AICallRecord is a redacted, persistable trace of a single model call, captured
// for cost control and explainability (CAL-036). It records sizes, latency, and
// a logical operation label — never prompt or response CONTENT — so candidate
// PII never reaches telemetry (token counts are approximated by character
// length until a tokenizer is wired).
type AICallRecord struct {
	Operation      string        // logical operation = the prompt id carried on the request
	PromptID       string        // registry prompt id (explicit; replaces substring guessing)
	PromptVersion  string        // registry prompt version — satisfies "version recorded per call"
	Model          string        // provider model id (or "dev")
	Latency        time.Duration // wall-clock time for the call
	PromptChars    int           // input size (proxy for input tokens)
	ResponseChars  int           // output size (proxy for output tokens)
	Failed         bool          // whether the call returned an error
	JSONFailure    bool          // structured-output (JSON) parse failure (CAL-137)
	Refusal        bool          // model refused the request (heuristic; CAL-137)
	GuardrailTrips []string      // guardrail categories that fired on this call (CAL-137)
	At             time.Time     // when the call started
}

// AICallRecorder persists AI-call traces. Implementations must be safe for
// concurrent use; recording must never block or fail the model call.
type AICallRecorder interface {
	Record(rec AICallRecord)
}
