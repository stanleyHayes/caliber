import type { Metric } from 'web-vitals';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { defaultReporter, reportWebVitals } from './vitals';

describe('reportWebVitals', () => {
  it('returns a no-op cleanup function', () => {
    const reporter = vi.fn();
    const unsubscribe = reportWebVitals(reporter);
    expect(typeof unsubscribe).toBe('function');
    expect(() => unsubscribe()).not.toThrow();
  });
});

describe('defaultReporter', () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
    vi.stubGlobal('console', { debug: vi.fn() });
  });
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  function makeMetric(overrides: Partial<Metric> = {}): Metric {
    return {
      name: 'LCP',
      value: 1200,
      rating: 'good',
      delta: 1200,
      id: 'v1',
      navigationType: 'navigate',
      entries: [],
      ...overrides,
    };
  }

  it('sends metrics to VITE_WEB_VITALS_ENDPOINT via sendBeacon', () => {
    vi.stubEnv('VITE_WEB_VITALS_ENDPOINT', 'https://projectcaliber.app/vitals');
    const sendBeacon = vi.fn().mockReturnValue(true);
    vi.stubGlobal('navigator', { sendBeacon });

    defaultReporter(makeMetric());

    expect(sendBeacon).toHaveBeenCalledTimes(1);
    const [url, payload] = sendBeacon.mock.calls[0] as [string, string];
    expect(url).toBe('https://projectcaliber.app/vitals');
    const body = JSON.parse(payload);
    expect(body).toMatchObject({
      name: 'LCP',
      value: 1200,
      rating: 'good',
      delta: 1200,
      id: 'v1',
      navigationType: 'navigate',
    });
  });

  it('falls back to fetch when sendBeacon is unavailable', async () => {
    vi.stubEnv('VITE_WEB_VITALS_ENDPOINT', 'https://projectcaliber.app/vitals');
    const fetch = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { sendBeacon: undefined });
    vi.stubGlobal('fetch', fetch);

    defaultReporter(makeMetric());

    await vi.waitFor(() => expect(fetch).toHaveBeenCalledTimes(1));
    const [url, init] = fetch.mock.calls[0] as [string, RequestInit];
    expect(url).toBe('https://projectcaliber.app/vitals');
    expect(init.method).toBe('POST');
    expect(init.headers).toEqual({ 'Content-Type': 'application/json' });
    expect(init.keepalive).toBe(true);
  });

  it('sends a Web Vitals event to Plausible when enabled', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    const plausible = vi.fn();
    window.plausible = plausible;

    defaultReporter(makeMetric({ name: 'CLS', value: 0.05, rating: 'needs-improvement' }));

    expect(plausible).toHaveBeenCalledWith('Web Vitals', {
      props: { metric: 'CLS', value: 0.05, rating: 'needs-improvement' },
    });
  });

  it('logs to console in dev when no sink is configured', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', '');
    vi.stubEnv('VITE_WEB_VITALS_ENDPOINT', '');
    vi.stubGlobal('navigator', {});

    defaultReporter(makeMetric());

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    expect((console as any).debug).toHaveBeenCalledWith('[web-vitals]', 'LCP', 1200, 'good');
  });
});
