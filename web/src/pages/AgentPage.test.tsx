import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError, type Application, type User, type WakeUpView } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { AgentPage } from './AgentPage';

type AdvanceResult = {
  mutate: ReturnType<typeof vi.fn>;
  isPending: boolean;
  isError: boolean;
  error: unknown;
  data?: { wakeUp: WakeUpView };
};
type ApplicationsResult = {
  isPending: boolean;
  isError: boolean;
  error: unknown;
  data?: { applications: Application[]; page?: { page: number; pageSize: number; totalItems: number; totalPages: number } };
};

let advanceResult: AdvanceResult;
let applicationsResult: ApplicationsResult;
const mutate = vi.fn();
const useApplicationsMock = vi.hoisted(() => vi.fn());
vi.mock('../query/agent', () => ({
  useTimeAdvance: () => advanceResult,
  useApplications: (...args: unknown[]) => useApplicationsMock(...args),
}));

const user: User = {
  id: 'cand-1',
  email: 'ama@example.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'Ama',
  createdAt: '2026-01-01T00:00:00Z',
};

beforeEach(() => {
  useAuthStore.setState({ user });
  mutate.mockReset();
  useApplicationsMock.mockReset();
  advanceResult = { mutate, isPending: false, isError: false, error: null };
  applicationsResult = { isPending: false, isError: false, error: null, data: { applications: [] } };
  useApplicationsMock.mockImplementation(() => applicationsResult);
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('AgentPage', () => {
  it('runs the agent overnight on demand', () => {
    render(<AgentPage />);
    fireEvent.click(screen.getByRole('button', { name: 'Run overnight' }));
    expect(mutate).toHaveBeenCalledTimes(1);
  });

  it('shows the wake-up summary after a run', () => {
    advanceResult = {
      mutate,
      isPending: false,
      isError: false,
      error: null,
      data: { wakeUp: { newMatches: 2, applicationsSubmitted: 1, screeningsCompleted: 0, employersInterested: 3, highlights: [] } },
    };
    render(<AgentPage />);
    expect(screen.getByText('While you were away')).toBeInTheDocument();
  });

  it('lists the agent-submitted applications', () => {
    applicationsResult = {
      isPending: false,
      isError: false,
      error: null,
      data: {
        applications: [
          {
            id: 'a1',
            roleId: 'role-1',
            candidateId: 'cand-1',
            source: 'APPLICATION_SOURCE_AGENT',
            tailoredSummary: 'Tailored to the payments role.',
            status: 'APPLICATION_STATUS_SUBMITTED',
          },
        ],
      },
    };
    render(<AgentPage />);
    expect(useApplicationsMock).toHaveBeenCalledWith('cand-1', 1, 20);
    expect(screen.getByText('Tailored to the payments role.')).toBeInTheDocument();
    expect(screen.getByText('by your agent')).toBeInTheDocument();
  });

  it('requests the selected server page when paginating applications', () => {
    applicationsResult = {
      isPending: false,
      isError: false,
      error: null,
      data: {
        applications: [
          {
            id: 'a1',
            roleId: 'role-1',
            candidateId: 'cand-1',
            source: 'APPLICATION_SOURCE_AGENT',
            tailoredSummary: 'Tailored to the payments role.',
            status: 'APPLICATION_STATUS_SUBMITTED',
          },
        ],
        page: { page: 1, pageSize: 20, totalItems: 41, totalPages: 3 },
      },
    };
    render(<AgentPage />);
    fireEvent.click(screen.getByRole('button', { name: /go to page 2/i }));
    expect(useApplicationsMock).toHaveBeenLastCalledWith('cand-1', 2, 20);
  });

  it('explains a 501 (agent needs the configured environment) plainly', () => {
    advanceResult = { mutate, isPending: false, isError: true, error: new ApiError(501, 'unimplemented') };
    render(<AgentPage />);
    expect(screen.getByText(/needs the configured environment/i)).toBeInTheDocument();
  });
});
