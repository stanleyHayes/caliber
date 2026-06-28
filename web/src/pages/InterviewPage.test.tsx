import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { InterviewQuestion, InterviewReportCard, User } from '../api/types';
import type { InterviewTurn } from '../hooks/useInterview';
import { useAuthStore } from '../stores/auth';
import { InterviewPage } from './InterviewPage';

type InterviewMock = {
  status: 'idle' | 'connecting' | 'asking' | 'submitting' | 'done' | 'error';
  question: InterviewQuestion | null;
  turns: InterviewTurn[];
  report: InterviewReportCard | null;
  error: string | null;
  start: ReturnType<typeof vi.fn>;
  answer: ReturnType<typeof vi.fn>;
  reset: ReturnType<typeof vi.fn>;
};

let interviewMock: InterviewMock;
vi.mock('../hooks/useInterview', () => ({ useInterview: () => interviewMock }));

const user: User = {
  id: 'cand-1',
  email: 'ama@example.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'Ama',
  createdAt: '2026-01-01T00:00:00Z',
};

function renderAt(path = '/interview') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <InterviewPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  useAuthStore.setState({ user });
  interviewMock = {
    status: 'idle',
    question: null,
    turns: [],
    report: null,
    error: null,
    start: vi.fn(),
    answer: vi.fn(),
    reset: vi.fn(),
  };
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('InterviewPage', () => {
  it('starts the interview for the role id from the query string', () => {
    renderAt('/interview?roleId=role-1');
    const start = screen.getByRole('button', { name: 'Start interview' });
    expect(start).toBeEnabled();
    fireEvent.click(start);
    expect(interviewMock.start).toHaveBeenCalledWith('role-1', 'cand-1');
  });

  it('disables start until a role id is provided', () => {
    renderAt('/interview');
    expect(screen.getByRole('button', { name: 'Start interview' })).toBeDisabled();
  });

  it('shows the current question while asking', () => {
    interviewMock.status = 'asking';
    interviewMock.question = { interviewId: 'iv1', ordinal: 1, text: 'Tell me about a Go service you built.', competencyTag: 'Go' };
    renderAt();
    expect(screen.getByText('Tell me about a Go service you built.')).toBeInTheDocument();
  });

  it('renders the report card and an option to run another interview when done', () => {
    interviewMock.status = 'done';
    interviewMock.report = {
      interviewId: 'iv1',
      roleId: 'role-1',
      candidateId: 'cand-1',
      verdict: 'INTERVIEW_VERDICT_ADVANCE',
      confidence: 'CONFIDENCE_HIGH',
      scores: [{ competency: 'Go', score: 4, evidence: 'built a ledger' }],
      recommendedNextStep: 'Advance to onsite.',
    };
    renderAt();
    expect(screen.getByText('Advance to onsite.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Run another interview' })).toBeInTheDocument();
  });

  it('surfaces a stream error with a way to start over', () => {
    interviewMock.status = 'error';
    interviewMock.error = 'The interview stalled — please try again.';
    renderAt();
    expect(screen.getByText('The interview stalled — please try again.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Start over' }));
    expect(interviewMock.reset).toHaveBeenCalledTimes(1);
  });
});
