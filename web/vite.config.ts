import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// SPA build; public marketing pages are prerendered at build time (CAL-121). In
// dev the /v1 API is proxied to the local gateway so the app calls it same-origin.
const proxyTarget = process.env.VITE_PROXY_TARGET ?? 'http://localhost:8080';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: { '/v1': proxyTarget },
  },
  build: { outDir: 'dist', sourcemap: true },
  // Bundle all dependencies into the SSR prerender bundle so the script runs
  // self-contained in Node without relying on any package's CJS entrypoints.
  ssr: { noExternal: true },
  test: {
    environment: 'jsdom',
    environmentOptions: { jsdom: { url: 'http://localhost/' } },
    globals: false,
    setupFiles: ['./src/test-setup.ts'],
    css: false,
    testTimeout: 20_000,
  },
});
