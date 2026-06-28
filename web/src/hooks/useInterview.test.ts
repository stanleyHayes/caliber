import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { InterviewEvent, InterviewReportCard } from '../api/types';
import { useInterview } from './useInterview';

// A controllable async stream: the test pushes events and ends it on demand,
// standing in for the real server-stream the hook consumes.
function makeStream() {
  const queue: InterviewEvent[] = [];
  let resolveNext: ((r: IteratorResult<InterviewEvent>) => void) | null = null;
  let ended = false;
  const push = (ev: InterviewEvent) => {
    if (resolveNext) {
      resolveNext({ value: ev, done: false });
      resolveNext = null;
    } else {
      queue.push(ev);
    }
  };
  const end = () => {
    ended = true;
    if (resolveNext) {
      resolveNext({ value: undefined, done: true });
      resolveNext = null;
    }
  };
  const iterable: AsyncIterable<InterviewEvent> = {
    [Symbol.asyncIterator]() {
      return {
        next() {
          if (queue.length > 0) return Promise.resolve({ value: queue.shift() as InterviewEvent, done: false });
          if (ended) return Promise.resolve({ value: undefined, done: true } as IteratorResult<InterviewEvent>);
          return new Promise<IteratorResult<InterviewEvent>>((r) => {
            resolveNext = r;
          });
        },
      };
    },
  };
  return { iterable, push, end };
}

const streamMock = vi.fn();
const submitMock = vi.fn();
vi.mock('../api/interview', () => ({
  streamInterview: (...args: unknown[]) => streamMock(...args),
  interviewApi: { submitAnswer: (...args: unknown[]) => submitMock(...args) },
}));

const question = (ordinal: number, interviewId = 'iv-1'): InterviewEvent => ({
  question: { interviewId, ordinal, text: `Question ${ordinal}?`, competencyTag: 'System design' },
});

const report: InterviewReportCard = {
  interviewId: 'iv-1',
  roleId: 'r1',
  candidateId: 'c1',
  verdict: 'INTERVIEW_VERDICT_ADVANCE',
  confidence: 'CONFIDENCE_HIGH',
  scores: [{ competency: 'System design', score: 4, evidence: 'designed a ledger' }],
  recommendedNextStep: 'Advance to onsite.',
};

beforeEach(() => {
  streamMock.mockReset();
  submitMock.mockReset();
  submitMock.mockResolvedValue({ accepted: true });
});
afterEach(() => vi.clearAllMocks());

describe('useInterview', () => {
  it('starts idle', () => {
    streamMock.mockReturnValue(makeStream().iterable);
    const { result } = renderHook(() => useInterview());
    expect(result.current.status).toBe('idle');
    expect(result.current.question).toBeNull();
    expect(result.current.turns).toEqual([]);
  });

  it('moves to "asking" when the first question arrives on the stream', async () => {
    const stream = makeStream();
    streamMock.mockReturnValue(stream.iterable);
    const { result } = renderHook(() => useInterview());

    act(() => result.current.start('r1', 'c1'));
    expect(result.current.status).toBe('connecting');

    await act(async () => {
      stream.push(question(1));
    });
    await waitFor(() => expect(result.current.status).toBe('asking'));
    expect(result.current.question?.text).toBe('Question 1?');
  });

  it('records the answered turn, submits it, and advances on the next question', async () => {
    const stream = makeStream();
    streamMock.mockReturnValue(stream.iterable);
    const { result } = renderHook(() => useInterview());

    act(() => result.current.start('r1', 'c1'));
    await act(async () => {
      stream.push(question(1));
    });
    await waitFor(() => expect(result.current.status).toBe('asking'));

    act(() => result.current.answer('I built a payments ledger.'));
    expect(result.current.status).toBe('submitting');
    // The captured turn carries the question, the answer, and its competency tag.
    expect(result.current.turns).toHaveLength(1);
    expect(result.current.turns[0]).toMatchObject({
      ordinal: 1,
      question: 'Question 1?',
      answer: 'I built a payments ledger.',
      competencyTag: 'System design',
    });
    // The answer was submitted against the interview id from the stream.
    expect(submitMock).toHaveBeenCalledWith('iv-1', 'I built a payments ledger.');

    await act(async () => {
      stream.push(question(2));
    });
    await waitFor(() => expect(result.current.status).toBe('asking'));
    expect(result.current.question?.ordinal).toBe(2);
  });

  it('finishes with the report card when it arrives', async () => {
    const stream = makeStream();
    streamMock.mockReturnValue(stream.iterable);
    const { result } = renderHook(() => useInterview());

    act(() => result.current.start('r1', 'c1'));
    await act(async () => {
      stream.push(question(1));
    });
    await act(async () => {
      stream.push({ reportCard: report });
    });

    await waitFor(() => expect(result.current.status).toBe('done'));
    expect(result.current.question).toBeNull();
    expect(result.current.report?.verdict).toBe('INTERVIEW_VERDICT_ADVANCE');
    expect(result.current.report?.recommendedNextStep).toBe('Advance to onsite.');
  });

  it('surfaces an error when the stream ends before any report card', async () => {
    const stream = makeStream();
    streamMock.mockReturnValue(stream.iterable);
    const { result } = renderHook(() => useInterview());

    act(() => result.current.start('r1', 'c1'));
    await act(async () => {
      stream.push(question(1));
      stream.end();
    });

    await waitFor(() => expect(result.current.status).toBe('error'));
    expect(result.current.error).toMatch(/ended unexpectedly/i);
  });

  it('reset returns the hook to its initial state', async () => {
    const stream = makeStream();
    streamMock.mockReturnValue(stream.iterable);
    const { result } = renderHook(() => useInterview());

    act(() => result.current.start('r1', 'c1'));
    await act(async () => {
      stream.push(question(1));
    });
    await waitFor(() => expect(result.current.status).toBe('asking'));

    act(() => result.current.reset());
    expect(result.current.status).toBe('idle');
    expect(result.current.question).toBeNull();
    expect(result.current.turns).toEqual([]);
  });
});
