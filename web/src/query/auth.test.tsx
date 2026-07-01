import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { authApi } from '../api/auth';
import type { AuthResponse, MeResponse, User } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { useLogin, useLogout, useMe, useRegister } from './auth';

const user: User = {
  id: 'u1',
  email: 'a@b.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'A',
  createdAt: '2026-01-01T00:00:00Z',
};
const authResponse: AuthResponse = {
  user,
  tokens: { accessToken: 'acc', refreshToken: 'ref', accessExpiresIn: 3600 },
};

vi.mock('../api/auth', () => ({
  authApi: {
    me: vi.fn(),
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  },
}));

function createWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe('useMe', () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
    localStorage.clear();
    vi.clearAllMocks();
  });
  afterEach(() => {
    useAuthStore.getState().clear();
  });

  it('fetches the current user and stores them in the auth store', async () => {
    useAuthStore.setState({ accessToken: 'acc' });
    vi.mocked(authApi.me).mockResolvedValue({ user } as MeResponse);

    const { result } = renderHook(() => useMe(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(useAuthStore.getState().user?.email).toBe('a@b.com');
    expect(result.current.data?.email).toBe('a@b.com');
  });

  it('stays disabled when there is no access token', () => {
    const { result } = renderHook(() => useMe(), { wrapper: createWrapper() });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useLogin', () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('stores the user and tokens on success', async () => {
    vi.mocked(authApi.login).mockResolvedValue(authResponse);

    const { result } = renderHook(() => useLogin(), { wrapper: createWrapper() });
    result.current.mutate({ email: 'a@b.com', password: 'secret' });

    await waitFor(() => expect(useAuthStore.getState().accessToken).toBe('acc'));
    expect(useAuthStore.getState().user?.email).toBe('a@b.com');
  });
});

describe('useRegister', () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('stores the user and tokens on success', async () => {
    vi.mocked(authApi.register).mockResolvedValue(authResponse);

    const { result } = renderHook(() => useRegister(), { wrapper: createWrapper() });
    result.current.mutate({ name: 'A', email: 'a@b.com', password: 'secret', role: 'USER_ROLE_CANDIDATE' });

    await waitFor(() => expect(useAuthStore.getState().accessToken).toBe('acc'));
    expect(useAuthStore.getState().user?.name).toBe('A');
  });
});

describe('useLogout', () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('calls logout with the refresh token, then clears the session and query cache', async () => {
    useAuthStore.getState().setSession(user, 'acc', 'ref');
    vi.mocked(authApi.logout).mockResolvedValue({});

    const { result } = renderHook(() => useLogout(), { wrapper: createWrapper() });
    result.current.mutate();

    await waitFor(() => expect(useAuthStore.getState().accessToken).toBeNull());
    expect(authApi.logout).toHaveBeenCalledWith('ref');
  });

  it('clears the session even when there is no refresh token', async () => {
    useAuthStore.getState().setSession(user, 'acc', null as unknown as string);

    const { result } = renderHook(() => useLogout(), { wrapper: createWrapper() });
    result.current.mutate();

    await waitFor(() => expect(useAuthStore.getState().accessToken).toBeNull());
    expect(authApi.logout).not.toHaveBeenCalled();
  });
});
