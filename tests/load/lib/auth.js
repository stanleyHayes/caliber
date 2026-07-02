import http from 'k6/http';
import { check } from 'k6';
import { urlFor } from '../config.js';

/**
 * Log in via the REST gateway and return token + user id.
 */
export function login(email, password) {
  const resp = http.post(
    urlFor('/v1/auth/login'),
    JSON.stringify({ email, password }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'auth_login' },
    }
  );

  check(resp, {
    'login status is 200': (r) => r.status === 200,
    'login returns access token': (r) => {
      const body = r.json();
      return body && body.tokens && body.tokens.accessToken && body.tokens.accessToken.length > 0;
    },
  });

  const body = resp.json();
  return {
    token: body.tokens.accessToken,
    userId: body.user.id,
    role: body.user.role,
  };
}

export function authHeader(token) {
  return {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
}
