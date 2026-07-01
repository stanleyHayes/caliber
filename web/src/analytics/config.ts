/**
 * Privacy-respecting analytics configuration (CAL-128).
 *
 * Everything is opt-in via environment variables. When no analytics provider is
 * configured, the build ships no tracking scripts and every tracking call is a
 * safe no-op.
 */
export type AnalyticsConfig = {
  /** Plausible data-domain. If empty, Plausible is disabled. */
  plausibleDomain: string;
  /** Plausible script URL. Override for self-hosted Plausible. */
  plausibleScriptUrl: string;
  /** Plausible event API URL. Override for self-hosted Plausible. */
  plausibleApiUrl: string;
  /** Optional endpoint to POST Web Vitals reports to. */
  webVitalsEndpoint: string;
  /** Optional Search Console meta-tag verification token. */
  searchConsoleVerification: string;
  /** Optional Search Console HTML file verification token (build-time). */
  searchConsoleHtmlFileToken: string;
};

function readVar(name: string): string {
  try {
    return import.meta.env?.[name] ?? '';
  } catch {
    return '';
  }
}

function trim(value: string): string {
  return value.trim();
}

export function getAnalyticsConfig(): AnalyticsConfig {
  return {
    plausibleDomain: trim(readVar('VITE_PLAUSIBLE_DOMAIN')),
    plausibleScriptUrl:
      trim(readVar('VITE_PLAUSIBLE_SCRIPT_URL')) || 'https://plausible.io/js/script.js',
    plausibleApiUrl: trim(readVar('VITE_PLAUSIBLE_API_URL')) || 'https://plausible.io/api/event',
    webVitalsEndpoint: trim(readVar('VITE_WEB_VITALS_ENDPOINT')),
    searchConsoleVerification: trim(readVar('VITE_SEARCH_CONSOLE_VERIFICATION')),
    searchConsoleHtmlFileToken: trim(readVar('VITE_SEARCH_CONSOLE_HTML_FILE_TOKEN')),
  };
}

export function isPlausibleEnabled(config: AnalyticsConfig = getAnalyticsConfig()): boolean {
  return Boolean(config.plausibleDomain);
}

export function areAnalyticsEnabled(config: AnalyticsConfig = getAnalyticsConfig()): boolean {
  return isPlausibleEnabled(config);
}
