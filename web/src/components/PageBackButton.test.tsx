import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it } from 'vitest';

import { PageBackButton } from './PageBackButton';

function renderWithRoutes(initialEntries: Parameters<typeof MemoryRouter>[0]['initialEntries'], initialIndex?: number) {
  return render(
    <MemoryRouter initialEntries={initialEntries} initialIndex={initialIndex}>
      <Routes>
        <Route path="/app" element={<div>Dashboard</div>} />
        <Route path="/roles" element={<div>Roles</div>} />
        <Route
          path="/profile"
          element={
            <div>
              <PageBackButton />
              <div>Profile</div>
            </div>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe('PageBackButton', () => {
  it('returns to the explicit source route when navigation state provides one', () => {
    renderWithRoutes([{ pathname: '/profile', state: { from: '/roles' } }]);

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));

    expect(screen.getByText('Roles')).toBeInTheDocument();
  });

  it('goes back through in-app route history when available', () => {
    renderWithRoutes(['/app', '/profile'], 1);

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));

    expect(screen.getByText('Dashboard')).toBeInTheDocument();
  });

  it('falls back to the dashboard when opened directly', () => {
    renderWithRoutes(['/profile']);

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));

    expect(screen.getByText('Dashboard')).toBeInTheDocument();
  });
});
