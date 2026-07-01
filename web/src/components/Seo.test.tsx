import { render, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { Seo } from './Seo';

describe('Seo', () => {
  it('emits a per-route title, description, and canonical (hoisted to <head> on React 19)', async () => {
    render(<Seo title="Sign in" description="Access your Caliber account." path="/login" />);
    await waitFor(() => expect(document.title).toBe('Sign in · Project Caliber'));

    const desc = document.head.querySelector('meta[name="description"]');
    expect(desc?.getAttribute('content')).toBe('Access your Caliber account.');
    const canonical = document.head.querySelector('link[rel="canonical"]');
    expect(canonical?.getAttribute('href')).toBe('https://projectcaliber.app/login');
  });

  it('keeps private routes out of the index', async () => {
    render(<Seo title="Dashboard" description="Your Talent Radar." path="/app" noindex />);
    await waitFor(() => {
      const robots = document.head.querySelector('meta[name="robots"]');
      expect(robots?.getAttribute('content')).toContain('noindex');
    });
  });

  it('emits hreflang alternate links for every supported locale on public pages', async () => {
    render(<Seo title="Sign in" description="Access your Caliber account." path="/login" />);
    await waitFor(() => {
      const links = Array.from(document.head.querySelectorAll('link[rel="alternate"]'));
      const hreflangs = links.map((l) => l.getAttribute('hreflang'));
      expect(hreflangs).toContain('en');
      expect(hreflangs).toContain('tw');
      expect(hreflangs).toContain('fr');
      expect(hreflangs).toContain('x-default');
      const enLink = links.find((l) => l.getAttribute('hreflang') === 'en');
      expect(enLink?.getAttribute('href')).toBe('https://projectcaliber.app/login?lng=en');
      const xDefault = links.find((l) => l.getAttribute('hreflang') === 'x-default');
      expect(xDefault?.getAttribute('href')).toBe('https://projectcaliber.app/login');
    });
  });

  it('does not emit hreflang links on noindex pages', async () => {
    render(<Seo title="Dashboard" description="Your Talent Radar." path="/app" noindex />);
    await waitFor(() => {
      const robots = document.head.querySelector('meta[name="robots"]');
      expect(robots?.getAttribute('content')).toContain('noindex');
    });
    const links = Array.from(document.head.querySelectorAll('link[rel="alternate"]'));
    expect(links).toHaveLength(0);
  });

  it('emits a Google Search Console verification meta tag when provided', async () => {
    render(
      <Seo
        title="Project Caliber"
        description="Talent intelligence."
        path="/"
        searchConsoleVerification="abc123"
      />,
    );
    await waitFor(() => {
      const meta = document.head.querySelector('meta[name="google-site-verification"]');
      expect(meta?.getAttribute('content')).toBe('abc123');
    });
  });
});
