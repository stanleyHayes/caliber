import { useCallback, useEffect, useRef, useState } from 'react';

import { interviewApi, streamInterview } from '../api/interview';
import type { InterviewQuestion, InterviewReportCard } from '../api/types';

export type InterviewStatus = 'idle' | 'connecting' | 'asking' | 'submitting' | 'done' | 'error';

export interface InterviewTurn {
  ordinal: number;
  question: string;
  answer: string;
  competencyTag: string;
}

interface InterviewState {
  status: InterviewStatus;
  question: InterviewQuestion | null;
  turns: InterviewTurn[];
  report: InterviewReportCard | null;
  error: string | null;
}

const initial: InterviewState = { status: 'idle', question: null, turns: [], report: null, error: null };
const STALL_MS = 30_000;

export function useInterview() {
  const [state, setState] = useState<InterviewState>(initial);
  const interviewId = useRef<string | null>(null);
  const abort = useRef<AbortController | null>(null);

  const start = useCallback((roleId: string, candidateId: string) => {
    abort.current?.abort();
    const controller = new AbortController();
    abort.current = controller;
    interviewId.current = null;
    setState({ ...initial, status: 'connecting' });
    // active() is false once this run is aborted or superseded by a newer run.
    const active = () => abort.current === controller && !controller.signal.aborted;

    void (async () => {
      let sawReport = false;
      try {
        for await (const ev of streamInterview(roleId, candidateId, controller.signal)) {
          if (!active()) {
            return;
          }
          if (ev.question) {
            interviewId.current = ev.question.interviewId;
            const question = ev.question;
            setState((s) => ({ ...s, status: 'asking', question }));
          } else if (ev.reportCard) {
            sawReport = true;
            const report = ev.reportCard;
            setState((s) => ({ ...s, status: 'done', question: null, report }));
          }
        }
        // Stream closed without a report card (backend/proxy dropped): surface it.
        if (active() && !sawReport) {
          setState((s) =>
            s.report ? s : { ...s, status: 'error', error: 'The interview ended unexpectedly. Please try again.' },
          );
        }
      } catch (e) {
        if (active()) {
          setState((s) => ({ ...s, status: 'error', error: e instanceof Error ? e.message : 'the interview stream failed' }));
        }
      }
    })();
  }, []);

  const answer = useCallback((text: string) => {
    const controller = abort.current;
    const id = interviewId.current;
    if (!controller || !id) {
      return;
    }
    setState((s) => {
      if (!s.question) {
        return s;
      }
      const turn: InterviewTurn = {
        ordinal: s.question.ordinal,
        question: s.question.text,
        answer: text,
        competencyTag: s.question.competencyTag,
      };
      return { ...s, status: 'submitting', question: null, turns: [...s.turns, turn] };
    });
    // The next question (or report card) arrives on the open stream.
    interviewApi.submitAnswer(id, text).catch((e: unknown) => {
      // Ignore late/stale rejections (restart, unmount, or after the report arrived).
      if (abort.current !== controller || controller.signal.aborted) {
        return;
      }
      controller.abort(); // the stream is now broken; stop it
      setState((s) =>
        s.report ? s : { ...s, status: 'error', error: e instanceof Error ? e.message : 'could not submit your answer' },
      );
    });
  }, []);

  const reset = useCallback(() => {
    abort.current?.abort();
    abort.current = null;
    interviewId.current = null;
    setState(initial);
  }, []);

  // Watchdog: a stalled stream (open but silent) should fail rather than hang.
  useEffect(() => {
    if (state.status !== 'connecting' && state.status !== 'submitting') {
      return undefined;
    }
    const timer = setTimeout(() => {
      setState((s) =>
        s.status === 'connecting' || s.status === 'submitting'
          ? { ...s, status: 'error', error: 'The interview stalled — please try again.' }
          : s,
      );
    }, STALL_MS);
    return () => clearTimeout(timer);
  }, [state.status]);

  useEffect(() => () => abort.current?.abort(), []);

  return { ...state, start, answer, reset };
}
