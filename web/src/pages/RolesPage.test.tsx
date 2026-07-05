import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { Role, User } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { RolesPage } from './RolesPage';

type RolesResult = {
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  data?: { roles: Role[]; page?: { page: number; pageSize: number; totalItems: number; totalPages: number } };
};
const useRolesMock = vi.hoisted(() => vi.fn());
let rolesResult: RolesResult;
vi.mock('../query/flow', () => ({ useRoles: (...args: unknown[]) => useRolesMock(...args) }));

const user: User = {
  id: 'emp-1',
  email: 'boss@acme.com',
  role: 'USER_ROLE_EMPLOYER',
  name: 'Boss',
  createdAt: '2026-01-01T00:00:00Z',
};

const role: Role = {
  id: 'role-1',
  employerId: 'emp-1',
  title: 'Senior Go Engineer',
  status: 'ROLE_STATUS_OPEN',
  createdAt: '2026-01-01T00:00:00Z',
  spec: {
    title: 'Senior Go Engineer',
    location: 'Accra',
    seniority: 'SENIORITY_SENIOR',
    availability: 'Full-time',
    responsibilities: [],
    mustHaves: ['Go'],
    niceToHaves: [],
    salaryBand: { currency: 'GHS', low: 0, high: 0 },
  },
  rubric: { competencies: [{ name: 'Go', weight: 1, mustHave: true }] },
};

function renderPage() {
  return render(
    <MemoryRouter>
      <RolesPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  useAuthStore.setState({ user });
  useRolesMock.mockReset();
  rolesResult = { isPending: false, isError: false, error: null, data: { roles: [] } };
  useRolesMock.mockImplementation(() => rolesResult);
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('RolesPage', () => {
  it('always offers a way to describe a new role', () => {
    renderPage();
    expect(screen.getByRole('link', { name: 'Describe a role' })).toHaveAttribute('href', '/roles/new');
  });

  it('prompts to create one when the employer has no roles', () => {
    renderPage();
    expect(screen.getByText(/No roles yet/i)).toBeInTheDocument();
  });

  it('lists a role with its seniority, location, and competency count', () => {
    rolesResult = { isPending: false, isError: false, error: null, data: { roles: [role] } };
    renderPage();
    expect(useRolesMock).toHaveBeenCalledWith('emp-1', 1, 20);
    expect(screen.getByRole('heading', { name: 'Senior Go Engineer' })).toBeInTheDocument();
    expect(screen.getByText('Senior')).toBeInTheDocument();
    expect(screen.getByText('Accra')).toBeInTheDocument();
    expect(screen.getByText('Full-time')).toBeInTheDocument();
    expect(screen.getByText('1 competency')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Interview' })).not.toBeInTheDocument();
  });

  it('requests the selected server page when paginating roles', () => {
    rolesResult = {
      isPending: false,
      isError: false,
      error: null,
      data: { roles: [role], page: { page: 1, pageSize: 20, totalItems: 45, totalPages: 3 } },
    };
    renderPage();
    fireEvent.click(screen.getByRole('button', { name: /go to page 2/i }));
    expect(useRolesMock).toHaveBeenLastCalledWith('emp-1', 2, 20);
  });

  it('surfaces a load error', () => {
    rolesResult = { isPending: false, isError: true, error: new Error('Could not reach the server') };
    renderPage();
    expect(screen.getByText('Could not reach the server')).toBeInTheDocument();
  });

  it('falls back to spec.title when role.title is empty', () => {
    const noTitle = { ...role, title: '' };
    rolesResult = { isPending: false, isError: false, error: null, data: { roles: [noTitle] } };
    renderPage();
    expect(screen.getByRole('heading', { name: 'Senior Go Engineer' })).toBeInTheDocument();
  });

  it('shows a generic message for a non-Error failure', () => {
    rolesResult = { isPending: false, isError: true, error: 'timeout' as unknown as Error };
    renderPage();
    expect(screen.getByText('Could not load roles.')).toBeInTheDocument();
  });
});
