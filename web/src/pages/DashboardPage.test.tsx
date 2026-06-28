import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { User, UserRole } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { DashboardPage } from './DashboardPage';

let meResult = { isPending: false };
vi.mock('../query/auth', () => ({ useMe: () => meResult }));

function userWith(role: UserRole): User {
  return { id: 'u1', email: 'u@example.com', role, name: 'Ama', createdAt: '2026-01-01T00:00:00Z' };
}

function renderPage() {
  return render(
    <MemoryRouter>
      <DashboardPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  meResult = { isPending: false };
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('DashboardPage', () => {
  it('greets an employer with role-specific next steps and CTAs', () => {
    useAuthStore.setState({ user: userWith('USER_ROLE_EMPLOYER') });
    renderPage();
    expect(screen.getByText('Employer')).toBeInTheDocument();
    expect(screen.getByText('Welcome, Ama.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Describe a role' })).toHaveAttribute('href', '/roles/new');
    expect(screen.getByRole('link', { name: 'Your roles' })).toHaveAttribute('href', '/roles');
    // No candidate CTAs for an employer.
    expect(screen.queryByRole('link', { name: 'Run your agent' })).not.toBeInTheDocument();
  });

  it('gives a candidate the passport / interview / agent CTAs', () => {
    useAuthStore.setState({ user: userWith('USER_ROLE_CANDIDATE') });
    renderPage();
    expect(screen.getByText('Candidate')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Set up your passport' })).toHaveAttribute('href', '/profile');
    expect(screen.getByRole('link', { name: 'Screening interview' })).toHaveAttribute('href', '/interview');
    expect(screen.getByRole('link', { name: 'Run your agent' })).toHaveAttribute('href', '/agent');
  });

  it('shows a skeleton while the session loads with no user yet', () => {
    meResult = { isPending: true };
    const { container } = renderPage();
    // No welcome heading yet — the skeleton placeholder is shown instead.
    expect(screen.queryByText(/Welcome/)).not.toBeInTheDocument();
    expect(container.querySelector('.MuiSkeleton-root')).not.toBeNull();
  });
});
