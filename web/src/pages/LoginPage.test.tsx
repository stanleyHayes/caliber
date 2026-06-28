import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LoginPage } from './LoginPage';

const mutate = vi.fn();
const loginState = { isError: false, error: null as Error | null, isPending: false };
vi.mock('../query/auth', () => ({ useLogin: () => ({ mutate, ...loginState }) }));

const navigateMock = vi.fn();
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useNavigate: () => navigateMock,
}));

function renderPage() {
  return render(
    <MemoryRouter>
      <LoginPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mutate.mockReset();
  navigateMock.mockReset();
  loginState.isError = false;
  loginState.error = null;
  loginState.isPending = false;
});
afterEach(() => vi.clearAllMocks());

describe('LoginPage', () => {
  it('submits the entered credentials and navigates to the app on success', () => {
    mutate.mockImplementation((_vars, opts?: { onSuccess?: () => void }) => opts?.onSuccess?.());
    renderPage();

    // MUI appends " *" to required-field labels, so match loosely.
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'ama@example.com' } });
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'sekret-pass' } });
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate.mock.calls[0][0]).toEqual({ email: 'ama@example.com', password: 'sekret-pass' });
    expect(navigateMock).toHaveBeenCalledWith('/app', { replace: true });
  });

  it('shows the error message and does not navigate when login fails', () => {
    loginState.isError = true;
    loginState.error = new Error('Invalid email or password');
    renderPage();

    expect(screen.getByText('Invalid email or password')).toBeInTheDocument();
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it('links to registration for new users', () => {
    renderPage();
    expect(screen.getByRole('link', { name: 'Create one' })).toHaveAttribute('href', '/register');
  });
});
