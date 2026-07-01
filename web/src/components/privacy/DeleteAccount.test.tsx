import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useAuthStore } from '../../stores/auth';
import { DeleteAccount } from './DeleteAccount';

const mutate = vi.fn();
const delState = { mutate, isPending: false, isError: false, error: null as Error | null };
vi.mock('../../query/privacy', () => ({ useDeleteMyData: () => delState }));

const navigateMock = vi.fn();
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useNavigate: () => navigateMock,
}));

function renderIt() {
  return render(
    <MemoryRouter>
      <DeleteAccount />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mutate.mockReset();
  navigateMock.mockReset();
  delState.isPending = false;
  delState.isError = false;
  delState.error = null;
  useAuthStore.setState({ accessToken: 'tok', refreshToken: 'ref' });
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('DeleteAccount', () => {
  it('requires typing DELETE before the irreversible action is enabled', () => {
    renderIt();
    fireEvent.click(screen.getByRole('button', { name: 'Delete my account' }));
    const confirm = screen.getByRole('button', { name: 'Delete everything' });
    expect(confirm).toBeDisabled();

    fireEvent.change(screen.getByLabelText(/Type DELETE/i), { target: { value: 'delete' } });
    expect(confirm).toBeEnabled(); // case-insensitive confirmation
  });

  it('erases the account, clears the session, and returns home on confirm', () => {
    mutate.mockImplementation((_v, opts?: { onSuccess?: () => void }) => opts?.onSuccess?.());
    renderIt();
    fireEvent.click(screen.getByRole('button', { name: 'Delete my account' }));
    fireEvent.change(screen.getByLabelText(/Type DELETE/i), { target: { value: 'DELETE' } });
    fireEvent.click(screen.getByRole('button', { name: 'Delete everything' }));

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState().accessToken).toBeNull(); // session cleared
    expect(navigateMock).toHaveBeenCalledWith('/', { replace: true });
  });

  it('closes the dialog and resets the confirmation when cancelled', () => {
    renderIt();
    fireEvent.click(screen.getByRole('button', { name: 'Delete my account' }));
    fireEvent.change(screen.getByLabelText(/Type DELETE/i), { target: { value: 'DELETE' } });
    expect(screen.getByRole('button', { name: 'Delete everything' })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    // Re-opening should show a fresh confirmation field.
    fireEvent.click(screen.getByText('Delete my account'));
    expect(screen.getByRole('button', { name: 'Delete everything' })).toBeDisabled();
  });

  it('does nothing if the confirmation word is wrong', () => {
    renderIt();
    fireEvent.click(screen.getByRole('button', { name: 'Delete my account' }));
    fireEvent.change(screen.getByLabelText(/Type DELETE/i), { target: { value: 'REMOVE' } });
    fireEvent.click(screen.getByRole('button', { name: 'Delete everything' }));

    expect(mutate).not.toHaveBeenCalled();
  });

  it('shows a server error message when deletion fails', () => {
    delState.isError = true;
    delState.error = new Error('service unavailable');
    renderIt();
    fireEvent.click(screen.getByRole('button', { name: 'Delete my account' }));
    expect(screen.getByText('service unavailable')).toBeInTheDocument();
  });
});
