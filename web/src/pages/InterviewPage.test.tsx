import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError, type InterviewQuestion, type InterviewReportCard, type Role, type User } from '../api/types';
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

type OpenRolesResult = {
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  data?: { roles: Role[]; page?: { page: number; pageSize: number; totalItems: number; totalPages: number } };
};

let interviewMock: InterviewMock;
let openRolesResult: OpenRolesResult;
vi.mock('../hooks/useInterview', () => ({ useInterview: () => interviewMock }));

vi.mock('../query/flow', () => ({
  useOpenRoles: () => openRolesResult,
}));

// The done state embeds ContestAssessment, which uses useRaiseContest.
vi.mock('../query/contest', () => ({
  useRaiseContest: () => ({ mutate: vi.fn(), isPending: false, isError: false, isSuccess: false, error: null }),
}));

const user: User = {
  id: 'cand-1',
  email: 'ama@example.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'Ama',
  createdAt: '2026-01-01T00:00:00Z',
};

const employer: User = {
  ...user,
  role: 'USER_ROLE_EMPLOYER',
};

const role: Role = {
  id: 'role-1',
  employerId: 'emp-1',
  title: 'Senior Go Engineer',
  status: 'ROLE_STATUS_OPEN',
  createdAt: '2026-01-01T00:00:00Z',
  spec: {
    title: 'Senior Go Engineer',
    location: 'Accra',
    seniority: 'SENIORITY_SENIOR',
    availability: 'Full-time',
    responsibilities: [],
    mustHaves: ['Go', 'Postgres'],
    niceToHaves: [],
    salaryBand: { currency: 'GHS', low: 0, high: 0 },
  },
  rubric: { competencies: [{ name: 'Go', weight: 1, mustHave: true }] },
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
  openRolesResult = { isPending: false, isError: false, error: null, data: { roles: [] } };
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('InterviewPage', () => {
  it('does not render the interview controls for a user without screen-self permission', () => {
    useAuthStore.setState({ user: employer });
    renderAt('/interview');
    expect(screen.queryByText('Choose a role')).not.toBeInTheDocument();
    expect(screen.queryByText(/missing permission/i)).not.toBeInTheDocument();
  });

  it('does not surface backend permission text if role listing is forbidden', () => {
    openRolesResult = { isPending: false, isError: true, error: new ApiError(403, 'auth: missing permission roles:manage') };
    renderAt('/interview');
    expect(screen.queryByText(/auth: missing permission/i)).not.toBeInTheDocument();
    expect(screen.queryByText('Choose a role')).not.toBeInTheDocument();
  });

  it('starts the interview by clicking a listed role', () => {
    openRolesResult = { isPending: false, isError: false, error: null, data: { roles: [role] } };
    renderAt('/interview');
    fireEvent.click(screen.getByRole('button', { name: 'Start interview for Senior Go Engineer' }));
    expect(interviewMock.start).toHaveBeenCalledWith('role-1', 'cand-1');
  });

  it('keeps role id entry as an intentional fallback', () => {
    renderAt('/interview');
    expect(screen.queryByLabelText('Role ID')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Use a role ID instead' }));
    expect(screen.getByLabelText('Role ID')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Start interview' })).toBeDisabled();
  });

  it('starts the interview for a linked role id from the query string', () => {
    renderAt('/interview?roleId=role-1');
    const start = screen.getByRole('button', { name: 'Start interview' });
    expect(start).toBeEnabled();
    fireEvent.click(start);
    expect(interviewMock.start).toHaveBeenCalledWith('role-1', 'cand-1');
  });

  it('starts the interview from a pasted role id fallback', () => {
    renderAt('/interview');
    fireEvent.click(screen.getByRole('button', { name: 'Use a role ID instead' }));
    fireEvent.change(screen.getByLabelText('Role ID'), { target: { value: 'role-2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Start interview' }));
    expect(interviewMock.start).toHaveBeenCalledWith('role-2', 'cand-1');
  });

  it('disables fallback start until a role id is provided', () => {
    renderAt('/interview');
    fireEvent.click(screen.getByRole('button', { name: 'Use a role ID instead' }));
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
