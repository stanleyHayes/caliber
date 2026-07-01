import { useEffect } from 'react';

import { reportWebVitals } from './vitals';

/**
 * Standalone Web Vitals hook. Use this in authenticated app routes when you
 * want Core Web Vitals reporting without Plausible pageview tracking.
 */
export function useWebVitals(): void {
  useEffect(() => {
    const unsubscribe = reportWebVitals();
    return () => {
      unsubscribe();
    };
  }, []);
}
