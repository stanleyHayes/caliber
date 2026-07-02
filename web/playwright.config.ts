import { defineConfig, devices } from '@playwright/test';

/**
 * Caliber e2e configuration.
 *
 * The suite expects the local stack to be running:
 *   make run-api            # http://localhost:8080
 *   cd web && npm run dev   # http://localhost:5173
 *
 * In CI the Docker Compose stack is started before the tests run.
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['html', { open: 'never' }], ['list']] : 'html',
  use: {
    baseURL: 'http://localhost:5173',
    // Pin the browser locale. The SPA's i18next detector falls back to the
    // browser's navigator.language and supports fr/tw, so on a runner whose OS
    // locale is non-English the app would render translated copy and every
    // hard-coded English selector ('Sign in', 'Welcome back', …) would fail.
    locale: 'en-US',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    contextOptions: {
      reducedMotion: 'reduce',
    },
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
