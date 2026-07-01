import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { trackEvent } from './track';

describe('trackEvent', () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
  });
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  it('is a no-op when Plausible is disabled', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', '');
    const plausible = vi.fn();
    window.plausible = plausible;
    trackEvent('Sign up');
    expect(plausible).not.toHaveBeenCalled();
  });

  it('calls window.plausible when enabled', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    const plausible = vi.fn();
    window.plausible = plausible;
    trackEvent('Sign up', { role: 'candidate' });
    expect(plausible).toHaveBeenCalledWith('Sign up', { props: { role: 'candidate' } });
  });

  it('drops undefined props so Plausible receives only real values', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    const plausible = vi.fn();
    window.plausible = plausible;
    trackEvent('View role', { roleId: 'r1', source: undefined });
    expect(plausible).toHaveBeenCalledWith('View role', { props: { roleId: 'r1' } });
  });

  it('does nothing if window.plausible is unavailable', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    delete window.plausible;
    expect(() => trackEvent('Sign up')).not.toThrow();
  });
});
