import { apiFetch } from './client';
import type { AuthResponse, LoginInput, MeResponse, RegisterInput } from './types';

export const authApi = {
  register: (input: RegisterInput) =>
    apiFetch<AuthResponse>('/v1/auth/register', { method: 'POST', auth: false, body: input }),
  login: (input: LoginInput) =>
    apiFetch<AuthResponse>('/v1/auth/login', { method: 'POST', auth: false, body: input }),
  me: () => apiFetch<MeResponse>('/v1/auth/me'),
  logout: (refreshToken: string) =>
    apiFetch<Record<string, never>>('/v1/auth/logout', {
      method: 'POST',
      auth: false,
      body: { refresh_token: refreshToken },
    }),
};
