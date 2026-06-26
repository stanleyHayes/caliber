import { useAuthStore } from '../stores/auth';
import { ApiError, type RefreshResponse } from './types';

const BASE = import.meta.env.VITE_API_URL ?? '';

interface FetchOpts {
  method?: string;
  body?: unknown;
  auth?: boolean;
}

let refreshInFlight: Promise<boolean> | null = null;

async function parseError(res: Response): Promise<ApiError> {
  let message = res.statusText || 'request failed';
  try {
    const body = (await res.json()) as { message?: string };
    if (body?.message) {
      message = body.message;
    }
  } catch {
    // non-JSON error body; keep the status text
  }
  return new ApiError(res.status, message);
}

// tryRefresh rotates the access token using the stored refresh token. Concurrent
// callers share a single in-flight refresh so the single-use refresh token is
// not consumed twice.
export async function tryRefresh(): Promise<boolean> {
  const { refreshToken } = useAuthStore.getState();
  if (!refreshToken) {
    return false;
  }
  refreshInFlight ??= (async () => {
    try {
      const res = await fetch(`${BASE}/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      if (!res.ok) {
        useAuthStore.getState().clear();
        return false;
      }
      const data = (await res.json()) as RefreshResponse;
      useAuthStore.getState().setTokens(data.tokens.accessToken, data.tokens.refreshToken);
      return true;
    } catch {
      useAuthStore.getState().clear();
      return false;
    } finally {
      refreshInFlight = null;
    }
  })();
  return refreshInFlight;
}

export async function apiFetch<T>(path: string, opts: FetchOpts = {}, allowRetry = true): Promise<T> {
  const { method = 'GET', body, auth = true } = opts;
  const token = useAuthStore.getState().accessToken;
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (auth && token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (res.status === 401 && auth && allowRetry && (await tryRefresh())) {
    return apiFetch<T>(path, opts, false);
  }
  if (!res.ok) {
    throw await parseError(res);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}
