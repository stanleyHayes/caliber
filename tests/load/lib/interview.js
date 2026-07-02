import { check, sleep } from 'k6';
import grpc from 'k6/net/grpc';
import { Stream } from 'k6/net/grpc';
import { getClient, metadata } from './grpc.js';

const ANSWER_TEXT =
  'I led a cross-functional team to rebuild our payments gateway in Go, ' +
  'cutting p99 latency from 400 ms to 80 ms and rolling it out to production over six weeks.';

// Matches the load-test overlay's CALIBER_INTERVIEW_MAX_QUESTIONS. The test
// caps questions so each interview fits inside the k6 scenario duration.
const EXPECTED_QUESTIONS = 2;

/**
 * Drive a full Flow B interview over gRPC server-streaming:
 *   1. Open a StartInterview stream.
 *   2. For every question event, submit an answer.
 *   3. Complete when the report card arrives.
 *
 * k6 delivers stream events asynchronously, so completion checks run inside the
 * event handlers rather than after a blocking wait.
 */
export function runInterview(candidateToken, candidateId, roleId) {
  const client = getClient();
  const params = metadata(candidateToken);
  const stream = new Stream(
    client,
    'caliber.v1.InterviewService/StartInterview',
    params
  );

  const state = {
    done: false,
    error: null,
    answersSubmitted: 0,
    gotReportCard: false,
  };

  stream.on('data', (event) => {
    if (event.question) {
      const q = event.question;
      const resp = client.invoke(
        'caliber.v1.InterviewService/SubmitAnswer',
        { interviewId: q.interviewId, answer: ANSWER_TEXT },
        params
      );
      check(resp, {
        'submit answer status is OK': (r) => r && r.status === grpc.StatusOK,
      });
      state.answersSubmitted++;
    } else if (event.reportCard) {
      state.gotReportCard = true;
      state.done = true;
      check(state, {
        'interview produced a report card': (s) => s.gotReportCard,
      });
    }
  });

  stream.on('error', (err) => {
    state.error = err;
    state.done = true;
  });

  stream.on('end', () => {
    state.done = true;
    check(state, {
      'interview completed without error': (s) => s.done && !s.error,
      'interview submitted expected answers': (s) => s.answersSubmitted >= EXPECTED_QUESTIONS,
    });
  });

  stream.write({
    roleId,
    candidateId,
    mode: 'INTERVIEW_MODE_TEXT',
  });

  // The server closes the stream once the report card is emitted. Yield so k6
  // can process stream events, but cap the wait so the VU does not hang.
  sleep(60);
}
