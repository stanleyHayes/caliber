package app

import "time"

// AICallRecord is a redacted, persistable trace of a single model call, captured
// for cost control and explainability (CAL-036). It records sizes, latency, and
// a logical operation label — never prompt or response CONTENT — so candidate
// PII never reaches telemetry (token counts are approximated by character
// length until a tokenizer is wired).
type AICallRecord struct {
	Operation     string        // logical operation, derived from the system prompt
	Model         string        // provider model id (or "dev")
	Latency       time.Duration // wall-clock time for the call
	PromptChars   int           // input size (proxy for input tokens)
	ResponseChars int           // output size (proxy for output tokens)
	Failed        bool          // whether the call returned an error
	At            time.Time     // when the call started
}

// AICallRecorder persists AI-call traces. Implementations must be safe for
// concurrent use; recording must never block or fail the model call.
type AICallRecorder interface {
	Record(rec AICallRecord)
}
