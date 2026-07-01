import { onCLS, onFCP, onINP, onLCP, onTTFB, type Metric } from 'web-vitals';

import { getAnalyticsConfig, isPlausibleEnabled } from './config';

export type WebVitalsReporter = (metric: Metric) => void;

/**
 * Default Web Vitals reporter. Sends metrics to:
 *  - the configured VITE_WEB_VITALS_ENDPOINT (if any), as a POST beacon;
 *  - Plausible as a custom event (if Plausible is enabled and window.plausible
 *    is available), using PII-free metric props.
 *
 * Falls back to console reporting in development when no endpoint is set, so
 * local builds still exercise the instrumentation without shipping data.
 */
export function defaultReporter(metric: Metric): void {
  const config = getAnalyticsConfig();

  if (config.webVitalsEndpoint && typeof navigator !== 'undefined') {
    const payload = JSON.stringify({
      name: metric.name,
      value: metric.value,
      rating: metric.rating,
      delta: metric.delta,
      id: metric.id,
      navigationType: metric.navigationType,
    });
    if (navigator.sendBeacon) {
      navigator.sendBeacon(config.webVitalsEndpoint, payload);
    } else {
      fetch(config.webVitalsEndpoint, {
        body: payload,
        headers: { 'Content-Type': 'application/json' },
        method: 'POST',
        keepalive: true,
      }).catch(() => {
        // Fail silently; vitals reporting must never break user-facing features.
      });
    }
  }

  if (isPlausibleEnabled(config) && typeof window !== 'undefined' && window.plausible) {
    window.plausible('Web Vitals', {
      props: {
        metric: metric.name,
        value: Math.round(metric.value * 1000) / 1000,
        rating: metric.rating,
      },
    });
  }

  if (
    !config.webVitalsEndpoint &&
    !isPlausibleEnabled(config) &&
    typeof console !== 'undefined' &&
    import.meta.env?.DEV
  ) {
    console.debug('[web-vitals]', metric.name, metric.value, metric.rating);
  }
}

/**
 * Subscribe to Core Web Vitals + supporting metrics. Safe to call multiple
 * times; `web-vitals` handles its own lifecycle. The reporter defaults to the
 * privacy-respecting endpoint/Plausible/console sink above.
 *
 * Returns a no-op cleanup because web-vitals v4 manages listeners internally
 * and does not expose per-metric unsubscribe handles.
 */
export function reportWebVitals(reporter: WebVitalsReporter = defaultReporter): () => void {
  onCLS(reporter);
  onFCP(reporter);
  onINP(reporter);
  onLCP(reporter);
  onTTFB(reporter);

  return () => {
    // no-op: web-vitals does not expose unsubscribe handles.
  };
}

export type { Metric };
