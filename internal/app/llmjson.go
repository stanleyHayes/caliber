package app

import (
	"context"
	"encoding/json"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// DefaultLLMAttempts is the default number of model attempts when decoding JSON:
// one initial call plus bounded re-asks on malformed output.
const DefaultLLMAttempts = 2

// jsonCorrection is appended to the prompt on a re-ask after unparseable output.
// It is an instruction from us (placed after any fenced untrusted content), not
// part of the third-party data.
const jsonCorrection = "\n\n[SYSTEM NOTICE] Your previous reply could not be parsed as JSON. " +
	"Reply with ONLY the JSON object specified above — no prose, no markdown fences."

// DecodeJSON is the structured-output enforcement boundary (CAL-031): it calls
// the model and decodes the reply into T, and when the reply is not valid JSON
// it re-asks up to attempts times, appending a corrective notice each round. A
// transport failure returns immediately as KindInternal; exhausting all attempts
// without parseable output returns KindInvalid. label prefixes error messages
// with the calling use-case for traceability.
//
// It only enforces that the output is well-formed JSON of the target shape;
// domain validity remains the job of the entity constructors the caller feeds.
//
//nolint:ireturn // generic decoder; T is the caller's concrete result type.
func DecodeJSON[T any](ctx context.Context, client LLMClient, req LLMRequest, attempts int, label string) (T, error) {
	var zero T
	if attempts < 1 {
		attempts = 1
	}
	req.ExpectJSON = true
	var lastErr error
	for range attempts {
		resp, err := client.Complete(ctx, req)
		if err != nil {
			return zero, kernel.Wrap(err, kernel.KindInternal, label+": model call failed")
		}
		var out T
		if uerr := json.Unmarshal([]byte(resp.Text), &out); uerr != nil {
			lastErr = uerr
			req.Prompt += jsonCorrection
			continue
		}
		return out, nil
	}
	return zero, kernel.Wrap(lastErr, kernel.KindInvalid, label+": model did not return valid JSON")
}
