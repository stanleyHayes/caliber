import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';

import { NotFoundPage } from './NotFoundPage';

describe('NotFoundPage', () => {
  it('explains the page is missing and offers a way home', () => {
    render(
      <MemoryRouter initialEntries={[{ pathname: '/404', state: { from: '/nope' } }]}>
        <NotFoundPage />
      </MemoryRouter>,
    );
    expect(
      screen.getByRole('heading', { name: "This page isn't in the evidence chain." }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Back to overview/ })).toHaveAttribute('href', '/');
    // Surfaces the originally-requested route forwarded via navigation state.
    expect(screen.getByText('/nope')).toBeInTheDocument();
  });
});
