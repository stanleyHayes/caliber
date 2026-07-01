import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { areAnalyticsEnabled, getAnalyticsConfig, isPlausibleEnabled } from './config';

describe('analytics config', () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
  });
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('returns empty defaults when no env vars are set', () => {
    const config = getAnalyticsConfig();
    expect(config.plausibleDomain).toBe('');
    expect(config.plausibleScriptUrl).toBe('https://plausible.io/js/script.js');
    expect(config.plausibleApiUrl).toBe('https://plausible.io/api/event');
    expect(config.webVitalsEndpoint).toBe('');
    expect(config.searchConsoleVerification).toBe('');
    expect(config.searchConsoleHtmlFileToken).toBe('');
  });

  it('reads VITE_PLAUSIBLE_DOMAIN and related vars', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    vi.stubEnv('VITE_PLAUSIBLE_SCRIPT_URL', 'https://analytics.example.com/js/script.js');
    vi.stubEnv('VITE_PLAUSIBLE_API_URL', 'https://analytics.example.com/api/event');
    vi.stubEnv('VITE_WEB_VITALS_ENDPOINT', 'https://projectcaliber.app/vitals');
    vi.stubEnv('VITE_SEARCH_CONSOLE_VERIFICATION', 'abc123');
    vi.stubEnv('VITE_SEARCH_CONSOLE_HTML_FILE_TOKEN', 'xyz789');

    const config = getAnalyticsConfig();
    expect(config.plausibleDomain).toBe('projectcaliber.app');
    expect(config.plausibleScriptUrl).toBe('https://analytics.example.com/js/script.js');
    expect(config.plausibleApiUrl).toBe('https://analytics.example.com/api/event');
    expect(config.webVitalsEndpoint).toBe('https://projectcaliber.app/vitals');
    expect(config.searchConsoleVerification).toBe('abc123');
    expect(config.searchConsoleHtmlFileToken).toBe('xyz789');
  });

  it('trims whitespace from env values', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', '  projectcaliber.app  ');
    expect(getAnalyticsConfig().plausibleDomain).toBe('projectcaliber.app');
  });

  it('reports analytics disabled by default', () => {
    expect(isPlausibleEnabled()).toBe(false);
    expect(areAnalyticsEnabled()).toBe(false);
  });

  it('reports analytics enabled when Plausible domain is set', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    expect(isPlausibleEnabled()).toBe(true);
    expect(areAnalyticsEnabled()).toBe(true);
  });

  it('reports analytics disabled when Plausible domain is only whitespace', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', '   ');
    expect(isPlausibleEnabled()).toBe(false);
  });
});
