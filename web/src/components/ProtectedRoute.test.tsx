import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { useAuthStore } from '../stores/auth';
import { ProtectedRoute } from './ProtectedRoute';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<div>Dashboard content</div>} />
        </Route>
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('ProtectedRoute', () => {
  it('redirects an unauthenticated visitor to /login', () => {
    renderAt('/dashboard');
    expect(screen.getByText('Login page')).toBeInTheDocument();
    expect(screen.queryByText('Dashboard content')).not.toBeInTheDocument();
  });

  it('renders the protected outlet when an access token is present', () => {
    useAuthStore.getState().setTokens('access-token', 'refresh-token');
    renderAt('/dashboard');
    expect(screen.getByText('Dashboard content')).toBeInTheDocument();
    expect(screen.queryByText('Login page')).not.toBeInTheDocument();
  });
});
