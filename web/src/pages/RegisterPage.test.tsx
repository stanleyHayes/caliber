import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RegisterPage } from './RegisterPage';

const mutate = vi.fn();
const registerState = { isError: false, error: null as Error | null, isPending: false };
vi.mock('../query/auth', () => ({ useRegister: () => ({ mutate, ...registerState }) }));

const navigateMock = vi.fn();
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useNavigate: () => navigateMock,
}));

function renderPage() {
  return render(
    <MemoryRouter>
      <RegisterPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mutate.mockReset();
  navigateMock.mockReset();
  registerState.isError = false;
  registerState.error = null;
  registerState.isPending = false;
});
afterEach(() => vi.clearAllMocks());

describe('RegisterPage', () => {
  it('submits the new account (defaulting to the employer role) and navigates to the app', () => {
    mutate.mockImplementation((_vars, opts?: { onSuccess?: () => void }) => opts?.onSuccess?.());
    renderPage();

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Ama Mensah' } });
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'ama@example.com' } });
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'a-very-long-password' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create account' }));

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate.mock.calls[0][0]).toEqual({
      name: 'Ama Mensah',
      email: 'ama@example.com',
      password: 'a-very-long-password',
      role: 'USER_ROLE_EMPLOYER',
    });
    expect(navigateMock).toHaveBeenCalledWith('/app', { replace: true });
  });

  it('surfaces a registration error without navigating', () => {
    registerState.isError = true;
    registerState.error = new Error('Email already in use');
    renderPage();

    expect(screen.getByText('Email already in use')).toBeInTheDocument();
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it('links back to sign-in for existing users', () => {
    renderPage();
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/login');
  });
});
