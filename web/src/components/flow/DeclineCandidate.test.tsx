import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { DeclineCandidate } from './DeclineCandidate';

const mutate = vi.fn();
const rejectState = { mutate, isPending: false, isError: false, isSuccess: false, error: null as Error | null };
vi.mock('../../query/flow', () => ({ useRecordRejection: () => rejectState }));

beforeEach(() => {
  mutate.mockReset();
  rejectState.isPending = false;
  rejectState.isError = false;
  rejectState.isSuccess = false;
  rejectState.error = null;
});
afterEach(() => vi.clearAllMocks());

describe('DeclineCandidate (human-approval gate)', () => {
  it('requires BOTH a reason and an explicit human confirmation before declining', () => {
    render(<DeclineCandidate roleId="role-1" candidateId="cand-1" />);
    fireEvent.click(screen.getByRole('button', { name: 'Decline candidate' }));

    const record = screen.getByRole('button', { name: 'Record decline' });
    expect(record).toBeDisabled();

    // A reason alone is not enough — the AI must not be able to auto-reject.
    fireEvent.change(screen.getByLabelText(/reason for declining/i), { target: { value: 'Needs deeper depth.' } });
    expect(record).toBeDisabled();

    // The explicit human confirmation unlocks it.
    fireEvent.click(screen.getByRole('checkbox'));
    expect(record).toBeEnabled();
  });

  it('records the decline with human_approved=true and the role + candidate', () => {
    render(<DeclineCandidate roleId="role-1" candidateId="cand-1" />);
    fireEvent.click(screen.getByRole('button', { name: 'Decline candidate' }));
    fireEvent.change(screen.getByLabelText(/reason for declining/i), { target: { value: 'Needs deeper depth.' } });
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByRole('button', { name: 'Record decline' }));

    expect(mutate).toHaveBeenCalledWith({
      roleId: 'role-1',
      candidateId: 'cand-1',
      reason: 'Needs deeper depth.',
      humanApproved: true,
    });
  });

  it('confirms the decline was human-approved and logged on success', () => {
    rejectState.isSuccess = true;
    render(<DeclineCandidate roleId="role-1" candidateId="cand-1" />);
    expect(screen.getByText(/human-approved/i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Decline candidate' })).not.toBeInTheDocument();
  });
});
