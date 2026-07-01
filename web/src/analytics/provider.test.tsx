import { render } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { AnalyticsProvider } from './provider';
import * as vitals from './vitals';

vi.mock('./vitals', () => ({ reportWebVitals: vi.fn().mockReturnValue(() => {}) }));
const reportWebVitals = vi.mocked(vitals.reportWebVitals);

describe('AnalyticsProvider', () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
    reportWebVitals.mockClear();
  });
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('renders children without a script when Plausible is disabled', () => {
    const { container, getByText } = render(
      <AnalyticsProvider>
        <div>content</div>
      </AnalyticsProvider>,
    );
    expect(getByText('content')).toBeInTheDocument();
    expect(container.querySelector('script')).not.toBeInTheDocument();
    expect(reportWebVitals).not.toHaveBeenCalled();
  });

  it('injects the Plausible script when VITE_PLAUSIBLE_DOMAIN is set', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    const { container } = render(
      <AnalyticsProvider>
        <div>content</div>
      </AnalyticsProvider>,
    );
    const script = container.querySelector('script');
    expect(script).toBeInTheDocument();
    expect(script).toHaveAttribute('defer');
    expect(script).toHaveAttribute('data-domain', 'projectcaliber.app');
    expect(script).toHaveAttribute('src', 'https://plausible.io/js/script.js');
    expect(script).toHaveAttribute('data-api', 'https://plausible.io/api/event');
    expect(reportWebVitals).toHaveBeenCalledTimes(1);
  });

  it('uses custom Plausible script and API URLs when provided', () => {
    vi.stubEnv('VITE_PLAUSIBLE_DOMAIN', 'projectcaliber.app');
    vi.stubEnv('VITE_PLAUSIBLE_SCRIPT_URL', 'https://analytics.example.com/js/script.js');
    vi.stubEnv('VITE_PLAUSIBLE_API_URL', 'https://analytics.example.com/api/event');
    const { container } = render(
      <AnalyticsProvider>
        <div>content</div>
      </AnalyticsProvider>,
    );
    const script = container.querySelector('script');
    expect(script).toHaveAttribute('src', 'https://analytics.example.com/js/script.js');
    expect(script).toHaveAttribute('data-api', 'https://analytics.example.com/api/event');
  });

  it('starts Web Vitals reporting without a script when only the endpoint is set', () => {
    vi.stubEnv('VITE_WEB_VITALS_ENDPOINT', 'https://projectcaliber.app/vitals');
    const { container } = render(
      <AnalyticsProvider>
        <div>content</div>
      </AnalyticsProvider>,
    );
    expect(container.querySelector('script')).not.toBeInTheDocument();
    expect(reportWebVitals).toHaveBeenCalledTimes(1);
  });
});
