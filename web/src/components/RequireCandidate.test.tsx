import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, describe, expect, it } from 'vitest';

import type { UserRole } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { RequireCandidate } from './RequireCandidate';

function renderAs(role: UserRole | undefined) {
  useAuthStore.setState({
    user: role ? { id: 'u1', email: 'x@y.z', role, name: 'X', createdAt: '2026-01-01T00:00:00Z' } : undefined,
  });
  return render(
    <MemoryRouter initialEntries={['/interview']}>
      <Routes>
        <Route element={<RequireCandidate />}>
          <Route path="/interview" element={<div>interview screen</div>} />
        </Route>
        <Route path="/app" element={<div>dashboard</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

afterEach(() => {
  useAuthStore.getState().clear();
});

describe('RequireCandidate', () => {
  it('lets a candidate reach the screening interview', () => {
    renderAs('USER_ROLE_CANDIDATE');
    expect(screen.getByText('interview screen')).toBeInTheDocument();
  });

  it('redirects a reviewer (employer) to the dashboard instead of a 403 dead-end', () => {
    renderAs('USER_ROLE_EMPLOYER');
    expect(screen.getByText('dashboard')).toBeInTheDocument();
    expect(screen.queryByText('interview screen')).not.toBeInTheDocument();
  });

  it('renders through while the role is still loading', () => {
    renderAs(undefined);
    expect(screen.getByText('interview screen')).toBeInTheDocument();
  });
});
