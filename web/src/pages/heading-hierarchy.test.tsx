import { render, screen } from '@testing-library/react';
import type { ComponentType } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { User } from '../api/types';
import { useAuthStore } from '../stores/auth';

import { AgentPage } from './AgentPage';
import { DashboardPage } from './DashboardPage';
import { EmployerFlowPage } from './EmployerFlowPage';
import { InterviewPage } from './InterviewPage';
import { LandingPage } from './LandingPage';
import { LoginPage } from './LoginPage';
import { NotFoundPage } from './NotFoundPage';
import { ProfilePage } from './ProfilePage';
import { RadarPage } from './RadarPage';
import { RegisterPage } from './RegisterPage';
import { RolesPage } from './RolesPage';

// Benign defaults so every page renders its main content (and its <h1>) rather
// than a loading skeleton. Hoisted so the vi.mock factories (also hoisted) can
// reference them.
const { query, mutation } = vi.hoisted(() => ({
  query: () => ({ isPending: false, isError: false, error: null, data: undefined }),
  mutation: () => ({ mutate: vi.fn(), isPending: false, isError: false, isSuccess: false, error: null }),
}));

vi.mock('../query/auth', () => ({ useMe: query, useLogin: mutation, useRegister: mutation }));
vi.mock('../query/flow', () => ({
  useRoles: query,
  useGenerateRole: mutation,
  useShortlist: query,
  useUpdateRole: mutation,
  useRecordRejection: mutation,
}));
vi.mock('../query/talent', () => ({ useProfile: query, useCreateProfile: mutation }));
vi.mock('../query/agent', () => ({ useTimeAdvance: mutation, useApplications: query }));
vi.mock('../query/radar', () => ({ usePool: query, useSupplyDemand: query, useTimeToShortlist: query }));
vi.mock('../query/contest', () => ({ useMyContests: query, useRaiseContest: mutation }));
vi.mock('../hooks/useInterview', () => ({
  useInterview: () => ({
    status: 'idle',
    question: null,
    turns: [],
    report: null,
    error: null,
    start: vi.fn(),
    answer: vi.fn(),
    reset: vi.fn(),
  }),
}));

class MockIntersectionObserver {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
  takeRecords = vi.fn().mockReturnValue([]);
}

const user: User = {
  id: 'u1',
  email: 'ama@example.com',
  role: 'USER_ROLE_EMPLOYER',
  name: 'Ama',
  createdAt: '2026-01-01T00:00:00Z',
};

const PAGES: [string, ComponentType][] = [
  ['LandingPage', LandingPage],
  ['LoginPage', LoginPage],
  ['RegisterPage', RegisterPage],
  ['NotFoundPage', NotFoundPage],
  ['RolesPage', RolesPage],
  ['ProfilePage', ProfilePage],
  ['AgentPage', AgentPage],
  ['DashboardPage', DashboardPage],
  ['RadarPage', RadarPage],
  ['EmployerFlowPage', EmployerFlowPage],
  ['InterviewPage', InterviewPage],
];

beforeEach(() => {
  useAuthStore.setState({ user });
  vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);
});
afterEach(() => {
  vi.unstubAllGlobals();
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('heading hierarchy (CAL-126)', () => {
  it.each(PAGES)('%s exposes exactly one top-level <h1>', (_name, Page) => {
    render(
      <MemoryRouter>
        <Page />
      </MemoryRouter>,
    );
    // Every page (and route) must have exactly one h1 — its main title — so the
    // document has a single, crawlable top-level heading and a valid hierarchy.
    const h1s = screen.getAllByRole('heading', { level: 1 });
    expect(h1s).toHaveLength(1);
    expect(h1s[0]).toHaveTextContent(/\S/);
  });
});
