import { areAnalyticsEnabled, getAnalyticsConfig, isPlausibleEnabled } from './config';

/**
 * Plausible's script exposes a global `plausible()` function when loaded. We
 * declare it minimally so TypeScript doesn't complain.
 */
declare global {
  interface Window {
    plausible?: (
      eventName: string,
      options?: { props?: Record<string, string | number | boolean> },
    ) => void;
  }
}

export type EventProps = Record<string, string | number | boolean | undefined>;

function sanitiseProps(props: EventProps): Record<string, string | number | boolean> {
  const out: Record<string, string | number | boolean> = {};
  for (const [key, value] of Object.entries(props)) {
    if (value === undefined) continue;
    out[key] = value;
  }
  return out;
}

/**
 * Track a privacy-respecting custom event. When Plausible is disabled this is a
 * no-op and never leaks data. Props with `undefined` values are dropped so the
 * event payload stays clean.
 */
export function trackEvent(eventName: string, props?: EventProps): void {
  const config = getAnalyticsConfig();
  if (!isPlausibleEnabled(config)) {
    return;
  }
  const payload = props ? sanitiseProps(props) : undefined;
  if (typeof window !== 'undefined' && typeof window.plausible === 'function') {
    window.plausible(eventName, payload ? { props: payload } : undefined);
  }
}

/**
 * Returns true when any analytics provider is enabled. Useful for components
 * that need to conditionally render analytics-related UI.
 */
export { areAnalyticsEnabled };
