import { afterEach, describe, expect, it, vi } from 'vitest';

import { importWithChunkReload, isChunkLoadError } from './lazyRoute';

describe('isChunkLoadError', () => {
  it('detects Vite dynamic-import failures', () => {
    expect(
      isChunkLoadError(
        new TypeError(
          'Failed to fetch dynamically imported module: https://calibergh.vercel.app/assets/DashboardPage-AC3v08no.js',
        ),
      ),
    ).toBe(true);
    expect(isChunkLoadError(new TypeError('Importing a module script failed.'))).toBe(true);
    expect(isChunkLoadError(new Error('boom'))).toBe(false);
  });
});

describe('importWithChunkReload', () => {
  afterEach(() => {
    sessionStorage.clear();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('returns the module on success and clears the reload guard', async () => {
    sessionStorage.setItem('caliber:chunk-reload', '1');
    const mod = { default: () => null };
    await expect(importWithChunkReload(async () => mod)).resolves.toBe(mod);
    expect(sessionStorage.getItem('caliber:chunk-reload')).toBeNull();
  });

  it('reloads once when a hashed chunk 404s after deploy', async () => {
    const reload = vi.fn();
    vi.stubGlobal('location', { ...window.location, reload });

    const pending = importWithChunkReload(async () => {
      throw new TypeError(
        'Failed to fetch dynamically imported module: https://calibergh.vercel.app/assets/DashboardPage-AC3v08no.js',
      );
    });

    await Promise.resolve();
    expect(reload).toHaveBeenCalledOnce();
    expect(sessionStorage.getItem('caliber:chunk-reload')).toBe('1');
    await expect(Promise.race([pending, Promise.resolve('pending')])).resolves.toBe('pending');
  });

  it('does not loop-reload when the guard is already set', async () => {
    sessionStorage.setItem('caliber:chunk-reload', '1');
    const reload = vi.fn();
    vi.stubGlobal('location', { ...window.location, reload });

    await expect(
      importWithChunkReload(async () => {
        throw new TypeError('Failed to fetch dynamically imported module: /assets/x.js');
      }),
    ).rejects.toThrow(/Failed to fetch dynamically imported module/);
    expect(reload).not.toHaveBeenCalled();
  });
});
