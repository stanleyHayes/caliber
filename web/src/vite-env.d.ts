/// <reference types="vite/client" />

interface ImportMetaEnv {
  // Base URL of the API. Empty in dev (the Vite proxy forwards /v1 same-origin);
  // set to the backend origin in production.
  readonly VITE_API_URL?: string;
}
interface ImportMeta {
  readonly env: ImportMetaEnv;
}

// Fontsource variable packages ship CSS only (no type declarations); declare
// them so their side-effect imports satisfy noUncheckedSideEffectImports.
declare module '@fontsource-variable/fraunces';
declare module '@fontsource-variable/outfit';
declare module '@fontsource-variable/jetbrains-mono';
