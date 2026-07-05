import { fireEvent, render, screen, within } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { Role, User } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { RoleDetailPage } from './RoleDetailPage';

const { useRoleMock, useDeleteRoleMock, useShortlistMock, useUpdateRoleMock, deleteMutate } = vi.hoisted(() => ({
  useRoleMock: vi.fn(),
  useDeleteRoleMock: vi.fn(),
  useShortlistMock: vi.fn(),
  useUpdateRoleMock: vi.fn(),
  deleteMutate: vi.fn(),
}));

vi.mock('../query/flow', () => ({
  useRole: (...args: unknown[]) => useRoleMock(...args),
  useDeleteRole: () => useDeleteRoleMock(),
  useShortlist: (...args: unknown[]) => useShortlistMock(...args),
  useUpdateRole: () => useUpdateRoleMock(),
}));

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
  title: 'Machine Learning Engineer',
  status: 'ROLE_STATUS_DRAFT',
  createdAt: '2026-01-01T00:00:00Z',
  spec: {
    title: 'Machine Learning Engineer',
    location: 'Remote',
    seniority: 'SENIORITY_MID',
    availability: 'Full-time',
    responsibilities: ['Build ranking models'],
    mustHaves: ['Python'],
    niceToHaves: ['MLOps'],
    salaryBand: { currency: 'GHS', low: 12000, high: 18000 },
  },
  rubric: {
    competencies: [
      { name: 'Python', weight: 0.6, mustHave: true },
      { name: 'Model evaluation', weight: 0.4, mustHave: false },
    ],
  },
};

function renderPage(initialEntry = '/roles/role-1') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/roles/:roleId" element={<RoleDetailPage />} />
        <Route path="/roles" element={<div>Roles index</div>} />
        <Route path="/app" element={<div>Dashboard</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  useAuthStore.setState({ user });
  useRoleMock.mockReset();
  useDeleteRoleMock.mockReset();
  useShortlistMock.mockReset();
  useUpdateRoleMock.mockReset();
  deleteMutate.mockReset();
  useRoleMock.mockReturnValue({ isPending: false, isError: false, error: null, data: { role } });
  useDeleteRoleMock.mockReturnValue({ mutate: deleteMutate, reset: vi.fn(), isPending: false, isError: false, error: null });
  useShortlistMock.mockReturnValue({ isPending: false, isError: false, error: null, data: undefined });
  useUpdateRoleMock.mockReturnValue({ mutate: vi.fn(), isPending: false, isError: false, error: null });
});

afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('RoleDetailPage', () => {
  it('loads and renders the selected role detail', () => {
    renderPage();

    expect(useRoleMock).toHaveBeenCalledWith('role-1', true);
    expect(screen.getByRole('heading', { name: 'Machine Learning Engineer', level: 1 })).toBeInTheDocument();
    expect(screen.getAllByText('Remote').length).toBeGreaterThan(0);
    expect(screen.getByText('2 competencies')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Refine role' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Archive role' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Scoring rubric' })).toBeInTheDocument();
  });

  it('opens the refinement editor from the detail view', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Refine role' }));

    expect(screen.getByRole('heading', { name: 'Refine spec & rubric' })).toBeInTheDocument();
    expect(screen.getByLabelText('Title')).toHaveValue('Machine Learning Engineer');
  });

  it('deletes the role and returns to the roles list', () => {
    deleteMutate.mockImplementation((_roleId: string, opts?: { onSuccess?: () => void }) => opts?.onSuccess?.());
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Archive role' }));
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Archive role' }));

    expect(deleteMutate).toHaveBeenCalledWith('role-1', expect.objectContaining({ onSuccess: expect.any(Function) }));
    expect(screen.getByText('Roles index')).toBeInTheDocument();
  });

  it('hides edit/delete for an employer who does not own the role', () => {
    useAuthStore.setState({ user: { ...user, id: 'emp-2' } }); // a different employer

    renderPage();

    // role.employerId is 'emp-1' — this reviewer isn't the owner, so the
    // destructive controls are hidden (mirrors the backend ownership guard).
    expect(screen.queryByRole('button', { name: 'Refine role' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Archive role' })).not.toBeInTheDocument();
  });

  it('redirects users without role-management permission', () => {
    useAuthStore.setState({ user: { ...user, role: 'USER_ROLE_CANDIDATE' } });

    renderPage();

    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(useRoleMock).toHaveBeenCalledWith('role-1', false);
  });
});
