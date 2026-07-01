import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { GenerateRoleResponse, Role, User } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { EmployerFlowPage } from './EmployerFlowPage';

const generateMutate = vi.fn();
const generateResult = { mutate: generateMutate, isPending: false, isError: false, error: null as Error | null };
vi.mock('../query/flow', () => ({
  useGenerateRole: () => generateResult,
  useShortlist: () => ({ isPending: false, isError: false, error: null, data: undefined, refetch: vi.fn() }),
  useUpdateRole: () => ({ mutate: vi.fn(), isPending: false }),
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
  title: 'Senior Go Engineer',
  status: 'ROLE_STATUS_OPEN',
  createdAt: '2026-01-01T00:00:00Z',
  spec: {
    title: 'Senior Go Engineer',
    location: 'Accra',
    seniority: 'SENIORITY_SENIOR',
    availability: 'Full-time',
    responsibilities: ['Own matching'],
    mustHaves: ['Go'],
    niceToHaves: [],
    salaryBand: { currency: 'GHS', low: 18000, high: 25000 },
  },
  rubric: { competencies: [{ name: 'Go', weight: 1, mustHave: true }] },
};
const generated: GenerateRoleResponse = { role, availableMatches: 3 };

function renderPage() {
  return render(
    <MemoryRouter>
      <EmployerFlowPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  useAuthStore.setState({ user });
  generateMutate.mockReset();
  generateResult.isPending = false;
  generateResult.isError = false;
  generateResult.error = null;
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('EmployerFlowPage', () => {
  it('gates generation until a brief is written', () => {
    renderPage();
    expect(screen.getByRole('button', { name: /Generate spec/ })).toBeDisabled();
  });

  it('generates a spec + rubric from the brief and surfaces the pool teaser', () => {
    generateMutate.mockImplementation((_vars, opts?: { onSuccess?: (d: GenerateRoleResponse) => void }) =>
      opts?.onSuccess?.(generated),
    );
    renderPage();

    fireEvent.change(screen.getByPlaceholderText(/senior Go backend engineer/i), {
      target: { value: 'Senior Go engineer in Accra' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Generate spec/ }));

    expect(generateMutate.mock.calls[0][0]).toEqual({ employerId: 'emp-1', freeText: 'Senior Go engineer in Accra' });
    // The structured result is shown: pool teaser, spec + rubric cards, shortlist.
    expect(screen.getByText('3 strong matches already in your pool.')).toBeInTheDocument();
    expect(screen.getByText('Scoring rubric')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Generate shortlist' })).toBeInTheDocument();
  });

  it('opens the rubric editor on refine', () => {
    generateMutate.mockImplementation((_vars, opts?: { onSuccess?: (d: GenerateRoleResponse) => void }) =>
      opts?.onSuccess?.(generated),
    );
    renderPage();
    fireEvent.change(screen.getByPlaceholderText(/senior Go backend engineer/i), { target: { value: 'a brief' } });
    fireEvent.click(screen.getByRole('button', { name: /Generate spec/ }));

    fireEvent.click(screen.getByRole('button', { name: /Refine spec/ }));
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeInTheDocument();
  });

  it('does not generate when the user is missing or the brief is empty', () => {
    useAuthStore.getState().clear();
    renderPage();
    fireEvent.change(screen.getByPlaceholderText(/senior Go backend engineer/i), { target: { value: 'a brief' } });
    fireEvent.click(screen.getByRole('button', { name: /Generate spec/ }));
    expect(generateMutate).not.toHaveBeenCalled();
  });

  it('shows a server error when generation fails', () => {
    generateResult.isError = true;
    generateResult.error = new Error('generation failed');
    renderPage();
    fireEvent.change(screen.getByPlaceholderText(/senior Go backend engineer/i), { target: { value: 'a brief' } });
    expect(screen.getByText('generation failed')).toBeInTheDocument();
  });

  it('welcomes a freshly generated spec even when the pool is empty', () => {
    generateMutate.mockImplementation((_vars, opts?: { onSuccess?: (d: GenerateRoleResponse) => void }) =>
      opts?.onSuccess?.({ role, availableMatches: 0 }),
    );
    renderPage();
    fireEvent.change(screen.getByPlaceholderText(/senior Go backend engineer/i), { target: { value: 'a brief' } });
    fireEvent.click(screen.getByRole('button', { name: /Generate spec/ }));
    expect(screen.getByText('Spec and rubric ready.')).toBeInTheDocument();
  });
});
