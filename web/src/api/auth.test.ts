import { beforeEach, describe, expect, it, vi } from 'vitest';

import { authApi } from './auth';
import { apiFetch } from './client';

vi.mock('./client', () => ({ apiFetch: vi.fn() }));

describe('authApi', () => {
  beforeEach(() => vi.clearAllMocks());

  it('registers a new user', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ user: { id: 'u1' } });
    const input = { name: 'A', email: 'a@b.com', password: 'secret', role: 'USER_ROLE_CANDIDATE' as const };
    await authApi.register(input);
    expect(apiFetch).toHaveBeenCalledWith('/v1/auth/register', { method: 'POST', auth: false, body: input });
  });

  it('logs in an existing user', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ user: { id: 'u1' } });
    const input = { email: 'a@b.com', password: 'secret' };
    await authApi.login(input);
    expect(apiFetch).toHaveBeenCalledWith('/v1/auth/login', { method: 'POST', auth: false, body: input });
  });

  it('fetches the current user', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ user: { id: 'u1' } });
    await authApi.me();
    expect(apiFetch).toHaveBeenCalledWith('/v1/auth/me');
  });

  it('logs out using the provided refresh token', async () => {
    vi.mocked(apiFetch).mockResolvedValue({});
    await authApi.logout('refresh-1');
    expect(apiFetch).toHaveBeenCalledWith('/v1/auth/logout', {
      method: 'POST',
      auth: false,
      body: { refresh_token: 'refresh-1' },
    });
  });
});
