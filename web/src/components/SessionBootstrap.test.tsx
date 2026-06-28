import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { User } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { SessionBootstrap } from './SessionBootstrap';

const tryRefreshMock = vi.fn();
const meMock = vi.fn();
vi.mock('../api/client', () => ({ tryRefresh: () => tryRefreshMock() }));
vi.mock('../api/auth', () => ({ authApi: { me: () => meMock() } }));

const user: User = {
  id: 'u1',
  email: 'ama@example.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'Ama Mensah',
  createdAt: '2026-01-01T00:00:00Z',
};

beforeEach(() => {
  tryRefreshMock.mockReset();
  meMock.mockReset();
  useAuthStore.getState().clear();
  localStorage.clear();
});
afterEach(() => {
  vi.clearAllMocks();
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('SessionBootstrap', () => {
  it('renders children immediately when an access token is already present', () => {
    useAuthStore.setState({ accessToken: 'access', refreshToken: 'refresh' });
    render(
      <SessionBootstrap>
        <div>App content</div>
      </SessionBootstrap>,
    );
    expect(screen.getByText('App content')).toBeInTheDocument();
    expect(tryRefreshMock).not.toHaveBeenCalled();
  });

  it('renders children immediately when there is no session to restore', () => {
    render(
      <SessionBootstrap>
        <div>App content</div>
      </SessionBootstrap>,
    );
    expect(screen.getByText('App content')).toBeInTheDocument();
    expect(tryRefreshMock).not.toHaveBeenCalled();
  });

  it('restores the session from a refresh token before revealing the app', async () => {
    useAuthStore.setState({ accessToken: null, refreshToken: 'refresh-only' });
    tryRefreshMock.mockResolvedValue(true);
    meMock.mockResolvedValue({ user });

    render(
      <SessionBootstrap>
        <div>App content</div>
      </SessionBootstrap>,
    );

    // While refreshing, the app is gated behind a skeleton (no content yet).
    expect(screen.queryByText('App content')).not.toBeInTheDocument();

    await waitFor(() => expect(screen.getByText('App content')).toBeInTheDocument());
    expect(tryRefreshMock).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState().user?.email).toBe('ama@example.com');
  });

  it('still reveals the app if the refresh fails (no usable session)', async () => {
    useAuthStore.setState({ accessToken: null, refreshToken: 'refresh-only' });
    tryRefreshMock.mockResolvedValue(false);

    render(
      <SessionBootstrap>
        <div>App content</div>
      </SessionBootstrap>,
    );

    await waitFor(() => expect(screen.getByText('App content')).toBeInTheDocument());
    expect(meMock).not.toHaveBeenCalled();
  });
});
