import react from '@vitejs/plugin-react';
import type { OutputAsset } from 'rollup';
import type { Plugin } from 'vite';
import { defineConfig } from 'vitest/config';

// SPA build; public marketing pages are prerendered at build time (CAL-121). In
// dev the /v1 API is proxied to the local gateway so the app calls it same-origin.
const proxyTarget = process.env.VITE_PROXY_TARGET ?? 'http://localhost:8080';

/**
 * Inject `<link rel="preload">` tags for the critical above-the-fold variable
 * fonts discovered in the build bundle. Preloading the latin subset of Fraunces
 * (display headings) and Outfit (body copy) removes the discovery delay for the
 * largest contentful paint text (CAL-125).
 */
function preloadFontsPlugin(): Plugin {
  const CRITICAL_FONT_PATTERNS = [
    /fraunces-latin-wght-normal/,
    /outfit-latin-wght-normal/,
  ];
  return {
    name: 'caliber-preload-fonts',
    apply: 'build',
    transformIndexHtml(html, ctx) {
      const bundle = ctx.bundle;
      if (!bundle) return html;
      const fonts = Object.values(bundle).filter(
        (chunk): chunk is OutputAsset =>
          chunk.type === 'asset' &&
          chunk.fileName.endsWith('.woff2') &&
          CRITICAL_FONT_PATTERNS.some((re) => re.test(chunk.fileName)),
      );
      if (fonts.length === 0) return html;
      const links = fonts
        .map(
          (font) =>
            `  <link rel="preload" href="/${font.fileName}" as="font" type="font/woff2" crossorigin>`,
        )
        .join('\n');
      return html.replace('</head>', `${links}\n</head>`);
    },
  };
}

export default defineConfig(({ isSsrBuild }) => ({
  plugins: [react(), preloadFontsPlugin()],
  server: {
    port: 5173,
    proxy: { '/v1': proxyTarget },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    // Split vendor libraries into cacheable chunks for the client build only.
    // SSR stays a single self-contained bundle so the prerender script can load
    // entry-server.js without resolving dynamic imports in Node.
    rollupOptions: isSsrBuild
      ? {}
      : {
          output: {
            manualChunks(id) {
              if (id.includes('node_modules')) {
                if (id.includes('@mui') || id.includes('@emotion')) return 'mui';
                if (id.includes('motion')) return 'motion';
                if (id.includes('@tanstack/react-query')) return 'query';
                if (id.includes('react-router') || id.includes('@remix-run')) return 'router';
                if (id.includes('react') || id.includes('scheduler')) return 'react';
                return 'vendor';
              }
            },
          },
        },
  },
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
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      thresholds: {
        statements: 80,
        branches: 80,
        functions: 80,
        lines: 80,
      },
      exclude: [
        'node_modules/',
        'dist/',
        'scripts/',
        '**/*.d.ts',
        'src/test-setup.ts',
        'src/main.tsx',
        'src/i18n/index.ts',
      ],
    },
  },
}));
