import { type ComponentType, type LazyExoticComponent, lazy } from 'react';

const RELOAD_GUARD_KEY = 'caliber:chunk-reload';

/**
 * True when a dynamic import failed because the hashed chunk is gone
 * (typical after a new Vercel deploy while a tab still holds the old bundle).
 */
export function isChunkLoadError(error: unknown): boolean {
  if (!error) return false;
  const msg = error instanceof Error ? error.message : String(error);
  return (
    /Failed to fetch dynamically imported module/i.test(msg) ||
    /Importing a module script failed/i.test(msg) ||
    /error loading dynamically imported module/i.test(msg) ||
    (error instanceof TypeError && /fetch/i.test(msg))
  );
}

/**
 * Runs a dynamic import; on a missing/stale hashed asset, reload once so the
 * browser picks up the new index + chunk map from the latest deploy.
 */
export async function importWithChunkReload<T>(importer: () => Promise<T>): Promise<T> {
  try {
    const mod = await importer();
    sessionStorage.removeItem(RELOAD_GUARD_KEY);
    return mod;
  } catch (error) {
    if (isChunkLoadError(error) && !sessionStorage.getItem(RELOAD_GUARD_KEY)) {
      sessionStorage.setItem(RELOAD_GUARD_KEY, '1');
      window.location.reload();
      // Keep Suspense pending until the reload navigates away.
      return new Promise<T>(() => undefined);
    }
    throw error;
  }
}

/** Lazy-load an authenticated route chunk with deploy-stale recovery. */
export function lazyRoute<T extends ComponentType<unknown>>(
  importer: () => Promise<{ default: T }>,
): LazyExoticComponent<T> {
  return lazy(() => importWithChunkReload(importer));
}
