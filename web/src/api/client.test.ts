import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useAuthStore } from '../stores/auth';
import { apiFetch, tryRefresh } from './client';
import { ApiError } from './types';

const user = { id: 'u1', email: 'a@b.com', role: 'USER_ROLE_CANDIDATE', name: 'A', createdAt: '2026-01-01T00:00:00Z' };
const tokens = { accessToken: 'acc', refreshToken: 'ref', accessExpiresIn: 3600 };

function mockResponse(status: number, body: unknown, opts?: { statusText?: string }) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: opts?.statusText ?? 'OK',
    headers: new Headers({ 'Content-Type': 'application/json' }),
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  } as Response);
}

describe('apiFetch', () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('returns parsed JSON on success', async () => {
    useAuthStore.getState().setSession(user, tokens.accessToken, tokens.refreshToken);
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(200, { id: '1' })));

    const result = await apiFetch<{ id: string }>('/v1/things');
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/v1/things'),
      expect.objectContaining({ headers: expect.objectContaining({ Authorization: 'Bearer acc' }) }),
    );
  });

  it('sends the body and method for POST requests', async () => {
    useAuthStore.getState().setSession(user, tokens.accessToken, tokens.refreshToken);
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(201, { ok: true })));

    await apiFetch('/v1/things', { method: 'POST', body: { name: 'x' } });
    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ name: 'x' }) }),
    );
  });

  it('omits the auth header for unauthenticated requests', async () => {
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(200, {})));

    await apiFetch('/v1/public', { auth: false });
    const [, opts] = (fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(opts.headers).not.toHaveProperty('Authorization');
  });

  it('returns undefined for a 204 response', async () => {
    useAuthStore.getState().setSession(user, tokens.accessToken, tokens.refreshToken);
    vi.stubGlobal(
      'fetch',
      vi.fn().mockReturnValue(
        Promise.resolve({ ok: true, status: 204, json: () => Promise.resolve({}) } as Response),
      ),
    );

    const result = await apiFetch('/v1/no-content');
    expect(result).toBeUndefined();
  });

  it('throws an ApiError with the server message on failure', async () => {
    useAuthStore.getState().setSession(user, tokens.accessToken, tokens.refreshToken);
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(422, { message: 'invalid' }, { statusText: 'Unprocessable' })));

    await expect(apiFetch('/v1/things')).rejects.toBeInstanceOf(ApiError);
    await expect(apiFetch('/v1/things')).rejects.toMatchObject({ status: 422, message: 'invalid' });
  });

  it('falls back to status text for non-JSON error bodies', async () => {
    useAuthStore.getState().setSession(user, tokens.accessToken, tokens.refreshToken);
    const res = { ok: false, status: 500, statusText: 'Server Error', json: () => Promise.reject(new Error('not json')) } as Response;
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(Promise.resolve(res)));

    await expect(apiFetch('/v1/things')).rejects.toMatchObject({ status: 500, message: 'Server Error' });
  });

  it('refreshes the token once on 401 and retries the request', async () => {
    useAuthStore.getState().setSession(user, tokens.accessToken, tokens.refreshToken);
    const refreshed = { accessToken: 'acc2', refreshToken: 'ref2', accessExpiresIn: 3600 };
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockReturnValueOnce(mockResponse(401, {}))
        .mockReturnValueOnce(mockResponse(200, { tokens: refreshed }))
        .mockReturnValueOnce(mockResponse(200, { id: '2' })),
    );

    const result = await apiFetch<{ id: string }>('/v1/things');
    expect(result).toEqual({ id: '2' });
    expect(useAuthStore.getState().accessToken).toBe('acc2');
  });

  it('does not retry a 401 when retry is disabled', async () => {
    useAuthStore.getState().setSession(user, tokens.accessToken, tokens.refreshToken);
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(401, {})));

    await expect(apiFetch('/v1/things', {}, false)).rejects.toBeInstanceOf(ApiError);
    expect(fetch).toHaveBeenCalledTimes(1);
  });
});

describe('tryRefresh', () => {
  beforeEach(() => {
    useAuthStore.getState().clear();
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('returns false when there is no refresh token', async () => {
    const result = await tryRefresh();
    expect(result).toBe(false);
  });

  it('updates tokens when the refresh endpoint succeeds', async () => {
    useAuthStore.getState().setSession(user, 'acc', 'ref');
    const refreshed = { accessToken: 'acc2', refreshToken: 'ref2', accessExpiresIn: 3600 };
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(200, { tokens: refreshed })));

    const result = await tryRefresh();
    expect(result).toBe(true);
    expect(useAuthStore.getState().accessToken).toBe('acc2');
  });

  it('clears the session when the refresh endpoint fails', async () => {
    useAuthStore.getState().setSession(user, 'acc', 'ref');
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(401, {})));

    const result = await tryRefresh();
    expect(result).toBe(false);
    expect(useAuthStore.getState().refreshToken).toBeNull();
  });

  it('shares a single in-flight refresh across concurrent callers', async () => {
    useAuthStore.getState().setSession(user, 'acc', 'ref');
    const refreshed = { accessToken: 'acc2', refreshToken: 'ref2', accessExpiresIn: 3600 };
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(mockResponse(200, { tokens: refreshed })));

    const [a, b] = await Promise.all([tryRefresh(), tryRefresh()]);
    expect(a).toBe(true);
    expect(b).toBe(true);
    expect(fetch).toHaveBeenCalledTimes(1);
  });
});
