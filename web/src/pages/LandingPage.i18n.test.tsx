import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { i18n } from '../i18n';
import { LandingPage } from './LandingPage';

class MockIntersectionObserver {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
  takeRecords = vi.fn().mockReturnValue([]);
}

function renderPage() {
  return render(
    <MemoryRouter>
      <I18nextProvider i18n={i18n}>
        <LandingPage />
      </I18nextProvider>
    </MemoryRouter>,
  );
}

describe('LandingPage i18n', () => {
  beforeEach(() => {
    vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);
  });

  afterEach(async () => {
    vi.unstubAllGlobals();
    await i18n.changeLanguage('en');
  });

  it('renders English copy by default', async () => {
    await i18n.changeLanguage('en');
    renderPage();
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Hire on evidence, not guesswork.');
    expect(screen.getByRole('link', { name: 'Get started' })).toBeInTheDocument();
  });

  it('renders Twi copy when the locale is switched', async () => {
    await i18n.changeLanguage('tw');
    renderPage();
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Fa nokware si adwuma, mmɔ nhyehyɛe.');
    expect(screen.getByRole('link', { name: 'Fie wo ho ase' })).toBeInTheDocument();
  });

  it('renders French copy when the locale is switched', async () => {
    await i18n.changeLanguage('fr');
    renderPage();
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Recrutez sur des preuves, pas sur des suppositions.');
    expect(screen.getByRole('link', { name: 'Commencer' })).toBeInTheDocument();
  });
});
