import { apiFetch, tryRefresh } from './client';
import { ApiError, type InterviewEvent } from './types';
import { useAuthStore } from '../stores/auth';

const BASE = import.meta.env.VITE_API_URL ?? '';

// streamInterview opens the StartInterview server-stream (a POST, so EventSource
// can't be used) and yields each event. grpc-gateway frames stream messages as
// newline-delimited JSON, each wrapped in {"result": <event>}.
export async function* streamInterview(
  roleId: string,
  candidateId: string,
  signal: AbortSignal,
): AsyncGenerator<InterviewEvent> {
  const res = await openStream(roleId, candidateId, signal);
  if (!res.body) {
    throw new ApiError(res.status, 'interview stream has no body');
  }
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      buffer += decoder.decode(value, { stream: true });
      let newline = buffer.indexOf('\n');
      while (newline >= 0) {
        const event = parseEventLine(buffer.slice(0, newline));
        buffer = buffer.slice(newline + 1);
        newline = buffer.indexOf('\n');
        if (event) {
          yield event;
        }
      }
    }
    // Flush a trailing event the server may have sent without a final newline.
    buffer += decoder.decode();
    const tail = parseEventLine(buffer);
    if (tail) {
      yield tail;
    }
  } finally {
    await reader.cancel().catch(() => undefined); // release the connection on abort/error/return
  }
}

// parseEventLine parses one stream line: it skips blanks and malformed JSON
// (rather than killing the stream) and throws on a gateway error frame.
function parseEventLine(line: string): InterviewEvent | null {
  const trimmed = line.trim();
  if (trimmed === '') {
    return null;
  }
  let parsed: { result?: InterviewEvent; error?: { message?: string } } & InterviewEvent;
  try {
    parsed = JSON.parse(trimmed) as typeof parsed;
  } catch {
    return null;
  }
  if (parsed.error) {
    throw new ApiError(0, parsed.error.message ?? 'the interview stream reported an error');
  }
  return parsed.result ?? parsed;
}

async function openStream(roleId: string, candidateId: string, signal: AbortSignal, retry = true): Promise<Response> {
  const token = useAuthStore.getState().accessToken;
  const res = await fetch(`${BASE}/v1/interviews:start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: JSON.stringify({ role_id: roleId, candidate_id: candidateId, mode: 'INTERVIEW_MODE_TEXT' }),
    signal,
  });
  if (res.status === 401 && retry && (await tryRefresh())) {
    return openStream(roleId, candidateId, signal, false);
  }
  if (!res.ok) {
    throw new ApiError(res.status, 'could not start the interview');
  }
  return res;
}

export const interviewApi = {
  submitAnswer: (interviewId: string, answer: string) =>
    apiFetch<{ accepted: boolean }>(`/v1/interviews/${encodeURIComponent(interviewId)}/answers`, {
      method: 'POST',
      body: { interview_id: interviewId, answer },
    }),
};
