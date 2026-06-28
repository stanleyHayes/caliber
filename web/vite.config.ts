import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// SPA build; public marketing pages are prerendered in a later story. In dev the
// /v1 API is proxied to the local gateway so the app calls it same-origin.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: { '/v1': 'http://localhost:8080' },
  },
  build: { outDir: 'dist', sourcemap: true },
  test: {
    environment: 'jsdom',
    globals: false,
    setupFiles: ['./src/test-setup.ts'],
    css: false,
  },
});
