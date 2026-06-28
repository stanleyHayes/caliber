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
type ApplicationsResult = { isPending: boolean; isError: boolean; error: unknown; data?: { applications: Application[] } };

let advanceResult: AdvanceResult;
let applicationsResult: ApplicationsResult;
const mutate = vi.fn();
vi.mock('../query/agent', () => ({
  useTimeAdvance: () => advanceResult,
  useApplications: () => applicationsResult,
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
  advanceResult = { mutate, isPending: false, isError: false, error: null };
  applicationsResult = { isPending: false, isError: false, error: null, data: { applications: [] } };
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
    expect(screen.getByText('Tailored to the payments role.')).toBeInTheDocument();
    expect(screen.getByText('by your agent')).toBeInTheDocument();
  });

  it('explains a 501 (agent needs the configured environment) plainly', () => {
    advanceResult = { mutate, isPending: false, isError: true, error: new ApiError(501, 'unimplemented') };
    render(<AgentPage />);
    expect(screen.getByText(/needs the configured environment/i)).toBeInTheDocument();
  });
});
