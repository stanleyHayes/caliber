import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// SPA build; public marketing pages are prerendered in a later story. In dev the
// /v1 API is proxied to the local gateway so the app calls it same-origin.
const proxyTarget = process.env.VITE_PROXY_TARGET ?? 'http://localhost:8080';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: { '/v1': proxyTarget },
  },
  build: { outDir: 'dist', sourcemap: true },
  test: {
    environment: 'jsdom',
    environmentOptions: { jsdom: { url: 'http://localhost/' } },
    globals: false,
    setupFiles: ['./src/test-setup.ts'],
    css: false,
    testTimeout: 20_000,
  },
});
