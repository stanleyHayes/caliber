import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// SPA build; public marketing pages are prerendered in a later story.
export default defineConfig({
  plugins: [react()],
  server: { port: 5173 },
  build: { outDir: 'dist', sourcemap: true },
});
