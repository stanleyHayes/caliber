import { beforeEach, describe, expect, it } from 'vitest';

import type { User } from '../api/types';
import { useAuthStore } from './auth';

const ama: User = {
  id: 'u1',
  email: 'ama@example.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'Ama Mensah',
  createdAt: '2026-01-01T00:00:00Z',
};

describe('useAuthStore', () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
    localStorage.clear();
  });

  it('setSession stores the user and both tokens', () => {
    useAuthStore.getState().setSession(ama, 'access1', 'refresh1');
    const s = useAuthStore.getState();
    expect(s.user?.email).toBe('ama@example.com');
    expect(s.accessToken).toBe('access1');
    expect(s.refreshToken).toBe('refresh1');
  });

  it('setTokens rotates the tokens but keeps the user', () => {
    useAuthStore.getState().setSession(ama, 'a', 'r');
    useAuthStore.getState().setTokens('a2', 'r2');
    const s = useAuthStore.getState();
    expect(s.accessToken).toBe('a2');
    expect(s.refreshToken).toBe('r2');
    expect(s.user?.email).toBe('ama@example.com');
  });

  it('clear resets the whole session', () => {
    useAuthStore.getState().setSession(ama, 'a', 'r');
    useAuthStore.getState().clear();
    const s = useAuthStore.getState();
    expect(s.user).toBeNull();
    expect(s.accessToken).toBeNull();
    expect(s.refreshToken).toBeNull();
  });

  it('persists only the refresh token — the access token and user stay in memory', () => {
    useAuthStore.getState().setSession(ama, 'access-secret', 'refresh-persisted');
    const persisted = JSON.parse(localStorage.getItem('caliber.auth') ?? '{}');
    expect(persisted.state.refreshToken).toBe('refresh-persisted');
    expect(persisted.state.accessToken).toBeUndefined();
    expect(persisted.state.user).toBeUndefined();
  });
});
