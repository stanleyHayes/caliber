/// <reference types="vite/client" />

interface ImportMetaEnv {
  // Base URL of the API. Empty in dev (the Vite proxy forwards /v1 same-origin);
  // set to the backend origin in production.
  readonly VITE_API_URL?: string;

  // Analytics & Search Console (CAL-128). All optional; disabled by default.
  /** Plausible data-domain. Enables Plausible pageview + event tracking. */
  readonly VITE_PLAUSIBLE_DOMAIN?: string;
  /** Plausible script URL. Override for self-hosted instances. */
  readonly VITE_PLAUSIBLE_SCRIPT_URL?: string;
  /** Plausible event API URL. Override for self-hosted instances. */
  readonly VITE_PLAUSIBLE_API_URL?: string;
  /** Endpoint to POST Web Vitals reports to. */
  readonly VITE_WEB_VITALS_ENDPOINT?: string;
  /** Google Search Console meta-tag verification content token. */
  readonly VITE_SEARCH_CONSOLE_VERIFICATION?: string;
  /** Google Search Console HTML file verification token (build-time). */
  readonly VITE_SEARCH_CONSOLE_HTML_FILE_TOKEN?: string;
}
interface ImportMeta {
  readonly env: ImportMetaEnv;
}

// Fontsource variable packages ship CSS only (no type declarations); declare
// them so their side-effect imports satisfy noUncheckedSideEffectImports.
declare module '@fontsource-variable/fraunces';
declare module '@fontsource-variable/outfit';
declare module '@fontsource-variable/jetbrains-mono';
