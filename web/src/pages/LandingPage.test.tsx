import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LandingPage } from './LandingPage';

// jsdom has no IntersectionObserver; motion's whileInView animations need it.
class MockIntersectionObserver {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
  takeRecords = vi.fn().mockReturnValue([]);
}

beforeEach(() => {
  vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);
});
afterEach(() => {
  vi.unstubAllGlobals();
});

function renderPage() {
  return render(
    <MemoryRouter>
      <LandingPage />
    </MemoryRouter>,
  );
}

describe('LandingPage', () => {
  it('pitches the three flagship capabilities', () => {
    renderPage();
    expect(screen.getByText('Explainable shortlists')).toBeInTheDocument();
    expect(screen.getByText('AI screening interviews')).toBeInTheDocument();
    expect(screen.getByText('An honest candidate agent')).toBeInTheDocument();
  });

  it('offers the primary calls to action', () => {
    renderPage();
    expect(screen.getByRole('link', { name: 'Get started' })).toHaveAttribute('href', '/register');
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/login');
  });
});
