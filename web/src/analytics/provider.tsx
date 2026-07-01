import { useEffect } from 'react';

import { getAnalyticsConfig, isPlausibleEnabled } from './config';
import { reportWebVitals } from './vitals';

/**
 * Injects the Plausible analytics script only when VITE_PLAUSIBLE_DOMAIN is set.
 * The script is loaded with `defer`, `data-api` defaults to Plausible's event
 * endpoint, and the domain is declared so outbound links/file downloads are not
 * tracked unless explicitly enabled.
 *
 * When disabled (the default), nothing is rendered and no request is made.
 */
export function AnalyticsProvider({ children }: { children: React.ReactNode }) {
  const config = getAnalyticsConfig();

  const shouldReportVitals =
    isPlausibleEnabled(config) || Boolean(config.webVitalsEndpoint);

  useEffect(() => {
    if (!shouldReportVitals) {
      return undefined;
    }

    const unsubscribe = reportWebVitals();
    return () => {
      unsubscribe();
    };
  }, [shouldReportVitals]);

  if (!isPlausibleEnabled(config)) {
    return <>{children}</>;
  }

  return (
    <>
      {children}
      <script
        defer
        data-domain={config.plausibleDomain}
        data-api={config.plausibleApiUrl}
        src={config.plausibleScriptUrl}
      />
    </>
  );
}
