import { ThemeProvider } from '@mui/material';
import { render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { User } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { theme } from '../theme/theme';
import { AppShell } from './AppShell';

const logoutMock = vi.fn();
vi.mock('../query/auth', () => ({ useLogout: () => ({ mutate: logoutMock, isPending: false }) }));

function renderShell(node: ReactNode = <AppShell />) {
  return render(
    <MemoryRouter>
      <ThemeProvider theme={theme} defaultMode="light">
        {node}
      </ThemeProvider>
    </MemoryRouter>,
  );
}

const user: User = {
  id: 'u1',
  email: 'ama@example.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'Ama Mensah',
  createdAt: '2026-01-01T00:00:00Z',
};

beforeEach(() => {
  logoutMock.mockReset();
  useAuthStore.getState().clear();
  localStorage.clear();
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }),
  );
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('AppShell', () => {
  it('shows the signed-out nav (sign in / get started) and points the brand home', () => {
    renderShell();
    expect(screen.getByRole('link', { name: 'Sign in' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Get started' })).toBeInTheDocument();
    expect(screen.queryByText('Radar')).not.toBeInTheDocument();
    expect(screen.queryByText(/Sign out/)).not.toBeInTheDocument();
    expect(screen.getByText('Caliber').closest('a')).toHaveAttribute('href', '/');
  });

  it('shows the signed-in nav (radar / sign out with the user name) and points the brand to the app', () => {
    useAuthStore.setState({ accessToken: 'access', user });
    renderShell();
    expect(screen.getByRole('link', { name: 'Radar' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign out (Ama Mensah)' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Sign in' })).not.toBeInTheDocument();
    expect(screen.getByText('Caliber').closest('a')).toHaveAttribute('href', '/app');
  });

  it('always exposes a skip-to-main-content link for keyboard users', () => {
    renderShell();
    expect(screen.getByText('Skip to main content').closest('a')).toHaveAttribute('href', '#main-content');
  });

  it('uses semantic landmarks: banner, primary navigation, and main content', () => {
    renderShell();
    expect(screen.getByRole('banner')).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Primary' })).toBeInTheDocument();
    const main = screen.getByRole('main');
    expect(main).toHaveAttribute('id', 'main-content');
  });
});
