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
});
