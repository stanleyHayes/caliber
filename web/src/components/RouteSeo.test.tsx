import { render, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';

import { RouteSeo } from './RouteSeo';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <RouteSeo />
    </MemoryRouter>,
  );
}

const robots = () => document.head.querySelector('meta[name="robots"]')?.getAttribute('content') ?? '';
// The JSON-LD <script> is not a head-hoisted tag (React 19 hoists title/meta/link),
// so it renders in the component tree — query the whole document for it.
const jsonLd = () => document.querySelector('script[type="application/ld+json"]')?.textContent ?? '';

describe('RouteSeo', () => {
  it('makes the public landing page indexable with Organization JSON-LD', async () => {
    renderAt('/');
    await waitFor(() => expect(document.title).toBe('Project Caliber'));
    expect(robots()).not.toContain('noindex');
    expect(jsonLd()).toContain('"@type":"Organization"');
  });

  it('gives the sign-in page its own indexable title without JSON-LD', async () => {
    renderAt('/login');
    await waitFor(() => expect(document.title).toBe('Sign in · Project Caliber'));
    expect(robots()).not.toContain('noindex');
    expect(jsonLd()).toBe('');
  });

  it('keeps authenticated app routes out of the index', async () => {
    renderAt('/agent');
    await waitFor(() => expect(robots()).toContain('noindex'));
    expect(jsonLd()).toBe('');
  });

  it('falls back to a noindex meta for unknown routes', async () => {
    renderAt('/some/unmapped/path');
    await waitFor(() => expect(document.title).toBe('Project Caliber'));
    expect(robots()).toContain('noindex');
  });
});
