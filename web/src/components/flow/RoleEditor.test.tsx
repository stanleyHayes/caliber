import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { Role } from '../../api/types';
import { RoleEditor } from './RoleEditor';

// useUpdateRole is mocked so the editor can be tested without a query client:
// mutate() invokes the supplied onSuccess with a returned role, as the real
// mutation would after the API responds.
const mutate = vi.fn();
let isPending = false;
const returnedRole = { id: 'r1', spec: { title: 'Saved' } } as unknown as Role;
vi.mock('../../query/flow', () => ({
  useUpdateRole: () => ({
    mutate: (vars: unknown, opts: { onSuccess: (d: { role: Role }) => void }) => {
      mutate(vars, opts);
      opts.onSuccess({ role: returnedRole });
    },
    get isPending() {
      return isPending;
    },
  }),
}));

const role: Role = {
  id: 'r1',
  employerId: 'e1',
  title: 'Backend Engineer',
  status: 'ROLE_STATUS_OPEN',
  createdAt: '2026-01-01T00:00:00Z',
  spec: {
    title: 'Backend Engineer',
    location: 'Accra',
    seniority: 'SENIORITY_MID',
    availability: 'Full-time',
    responsibilities: ['Build services'],
    mustHaves: ['Go'],
    niceToHaves: [],
    salaryBand: { currency: 'GHS', low: 1000, high: 5000 },
  },
  rubric: { competencies: [{ name: 'Go', weight: 0.6, mustHave: true }] },
};

beforeEach(() => {
  mutate.mockReset();
  isPending = false;
});
afterEach(() => vi.clearAllMocks());

describe('RoleEditor', () => {
  it('prefills the form from the role spec and preserves untouched fields on save', () => {
    const onSaved = vi.fn();
    render(<RoleEditor role={role} onSaved={onSaved} onCancel={vi.fn()} />);

    expect(screen.getByLabelText('Title')).toHaveValue('Backend Engineer');
    expect(screen.getByLabelText('Location')).toHaveValue('Accra');

    // Edit the title, then save.
    fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'Senior Backend Engineer' } });
    fireEvent.click(screen.getByText('Save changes'));

    expect(mutate).toHaveBeenCalledTimes(1);
    const [vars] = mutate.mock.calls[0];
    expect(vars.roleId).toBe('r1');
    expect(vars.spec.title).toBe('Senior Backend Engineer');
    // An untouched field survives the edit (the full spec is held, not just title).
    expect(vars.spec.mustHaves).toEqual(['Go']);
    expect(vars.rubric.competencies).toEqual([{ name: 'Go', weight: 0.6, mustHave: true }]);
    // onSuccess routes the returned role back to the caller.
    expect(onSaved).toHaveBeenCalledWith(returnedRole);
  });

  it('disables Save when every rubric weight is zero (nothing to rank on)', () => {
    const zeroWeighted: Role = {
      ...role,
      rubric: { competencies: [{ name: 'Go', weight: 0, mustHave: true }] },
    };
    render(<RoleEditor role={zeroWeighted} onSaved={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByText('Save changes').closest('button')).toBeDisabled();
  });

  it('invokes onCancel without saving', () => {
    const onCancel = vi.fn();
    render(<RoleEditor role={role} onSaved={vi.fn()} onCancel={onCancel} />);
    fireEvent.click(screen.getByText('Cancel'));
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(mutate).not.toHaveBeenCalled();
  });
});
